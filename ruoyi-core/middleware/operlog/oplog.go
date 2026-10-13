// Package operlog 提供操作日志注解（装饰器）。
//
// 落库实现不在本包：pkg 不依赖 internal，故这里只组装 Event，由 internal/system
// 在启动时经 Init 反向注册 Recorder 消费。
//
// 与原实现的几处有意差异：
//   - Method 取 gin 的 HandlerName（无 AOP 切点，拿不到"类名.方法名"）；
//   - OperParam 在 handler 之前采集——Go 的 body 是一次性流，handler 消费后再读就没了；
//   - 响应体边写边抄而非事后补发，panic 时半截响应的行为与不挂本中间件一致。
package operlog

import (
	"context"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/enum"
)

// 各字段落库长度上限，与 sys_oper_log 列宽对应；超长会让整条日志插入失败。
const (
	maxURLLength       = 255 // oper_url varchar(255)
	maxClientKeyLength = 32  // client_key / device_type varchar(32)
	maxContentLength   = 3800
	maxNameLength      = 50 // oper_name / dept_name / browser / os varchar(50)
	maxIPLength        = 128
)

// msgPanic panic 时写入 ErrorMsg 的占位文案。不 recover 取真实值：re-panic 会把
// Recover 那边的栈顶挪到本包，排障看不到出事行。真实值与栈由 Recover 记录。
const msgPanic = "系统异常"

// Event 操作日志事件。OperLocation 留空由 Recorder 侧补（IP 归属地查询不该占请求线程）。
type Event struct {
	Title         string
	BusinessType  int
	Method        string
	RequestMethod string
	OperatorType  int
	OperName      string
	UserID        int64
	DeptID        int64
	DeptName      string
	ClientKey     string
	DeviceType    string
	Browser       string
	OS            string
	OperURL       string
	OperIP        string
	OperParam     string
	JSONResult    string
	Status        int
	ErrorMsg      string
	CostTime      int64
}

// Recorder 消费操作日志事件，由 internal/system 注册。实现方异步落库并自行兜错，本包不等待不检查返回。
type Recorder func(ctx context.Context, evt *Event)

// recorder 包级 Recorder。只在 Init 写一次、每次请求都读，atomic.Pointer 足够且免去读写锁。
var recorder atomic.Pointer[Recorder]

// Init 注册落库实现，须在 database.Init 之后调用（Recorder 内部要写库）。
func Init(r Recorder) {
	if r == nil {
		panic("operlog: Recorder 不能为空")
	}
	recorder.Store(&r)
}

// get 取已注册的 Recorder，未注册返回 nil。不 panic：日志缺失不该让请求失败，漏注册由告警暴露。
func get() Recorder {
	p := recorder.Load()
	if p == nil {
		return nil
	}
	return *p
}

// options Log 的可选项。
type options struct {
	operatorType     enum.OperatorType
	saveRequestData  bool
	saveResponseData bool
	excludeParams    []string
}

// Option 配置项。
type Option func(*options)

// WithOperatorType 指定操作人类别，默认 OperatorTypeManage。
func WithOperatorType(t enum.OperatorType) Option {
	return func(o *options) { o.operatorType = t }
}

// WithoutRequestData 不记录请求参数。
func WithoutRequestData() Option {
	return func(o *options) { o.saveRequestData = false }
}

// WithoutResponseData 不记录响应参数。
func WithoutResponseData() Option {
	return func(o *options) { o.saveResponseData = false }
}

// WithExcludeParams 追加不记录的请求参数名。constant.ExcludeProperties 里的密码类字段无需重复。
func WithExcludeParams(names ...string) Option {
	return func(o *options) { o.excludeParams = append(o.excludeParams, names...) }
}

// Log 操作日志注解。title 为模块名，businessType 为业务类型。
//
// 须排在鉴权之后：被拒的请求在鉴权阶段就返回，进不到本中间件。
func Log(title string, businessType enum.BusinessType, opts ...Option) gin.HandlerFunc {
	o := &options{
		operatorType:     enum.OperatorTypeManage,
		saveRequestData:  true,
		saveResponseData: true,
	}
	for _, opt := range opts {
		opt(o)
	}

	return func(c *gin.Context) {
		run(c, title, businessType, o)
	}
}
