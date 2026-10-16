// Package repeatsubmit 提供防重复提交注解（装饰器）。
//
// 判定逻辑（参考美团 GTIS 防重）：
//   - 进 handler 前用 SETNX 抢锁，抢不到即判为重复提交并拦截；
//   - handler 成功（响应 code=200）则保留键，interval 内的重复请求都被挡住；
//   - handler 失败或 panic 则删键，允许立刻重试。
package repeatsubmit

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

const (
	defaultInterval = 5 * time.Second // 默认防重间隔
	minInterval     = time.Second     // 最小间隔，更小则在注册期 panic
	defaultMessage  = "不允许重复提交，请稍候再试" // 默认提示文案
)

// defaultTokenName 未配置时读取 token 的请求头名。
const defaultTokenName = "Authorization"

// maxFingerprintBody 参与指纹计算的请求体上限，超出部分不读。
const maxFingerprintBody = 1 << 20 // 1MB

// codeSuccess 业务成功码，与 ruoyi-core/model/response.CodeSuccess 同值。
// 就地定义避免引 ruoyi-core/model/response 包（仅用此一个常量）。
const codeSuccess = 200

// codeFail 业务失败码，与 ruoyi-core/model/response.CodeFail 同值；防重触发时回此码。
// 就地定义避免引 ruoyi-core/model/response 包。
const codeFail = 500

// failBody 失败响应体，形态同 ruoyi-core/model/response.R（Data 恒为 null）。
// 用结构体而非 fmt 拼串：message 可由调用方自定义，含引号时拼串会产出非法 JSON。
type failBody struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

// repeatSubmitKey 防重缓存键前缀。"global:" 前缀沿用全局 redis key 约定。
const repeatSubmitKey = "global:repeat_submit:"

type options struct {
	interval time.Duration // 防重间隔
	message  string        // 触发时的提示文案
}

// newOptions 构造防重配置。interval 为 0 取默认值，message 为空取默认词条；interval < 1 秒 panic。
func newOptions(interval time.Duration, message string) *options {
	if interval == 0 {
		interval = defaultInterval
	}
	if message == "" {
		message = defaultMessage
	}
	if interval < minInterval {
		panic("repeatsubmit: 重复提交间隔时间不能小于 1 秒")
	}
	return &options{interval: interval, message: message}
}

// Submitter 防重提交器。
type Submitter struct {
	tokenName string          // 读取 token 的请求头名
	rdb       *goredis.Client // 防重键所用的 Redis 客户端，由 Init 注入
}

// defaultSubmitter 包级默认实例。用 atomic.Pointer 取代 mutex：只在 Init 写一次、
// 每次请求都读，指针的原子读写足够且去掉了读写锁互斥。
var defaultSubmitter atomic.Pointer[Submitter]

// Init 构造包级默认实例。rdb 为防重键所用的 Redis 客户端，须在 redis.Init 之后传入 redis.Client()；
// tokenName 取自调用方（模块装配时从 satoken 配置取），为空时兜底 defaultTokenName。
// 必须先于路由注册，否则注册限重路由那一刻 panic，比拖到首次请求才崩更易定位。
func Init(rdb *goredis.Client, tokenName string) {
	// 移除了原先对包级 redis.Client() 的兜底，nil 必须在此拦下，否则要到请求时才 NPE，难定位。
	if rdb == nil {
		panic("repeatsubmit: Init 收到 nil redis 客户端，请先 redis.Init 再传入 redis.Client()")
	}
	if tokenName == "" {
		tokenName = defaultTokenName
	}
	s := &Submitter{tokenName: tokenName, rdb: rdb}
	defaultSubmitter.Store(s)
	log.Printf("repeatsubmit 已就绪: tokenName=%s", s.tokenName)
}

// RepeatSubmit 防重复提交注解。interval 为 0、message 为空时取默认值；interval < 1 秒注册期 panic。
//
// 须排在 apiencrypt 之后：取到的请求体须是解密后的明文，指纹才稳定
// （密文每次随机 AES 密钥，同样入参会算出不同指纹，防重直接失效）。
func RepeatSubmit(interval time.Duration, message string) gin.HandlerFunc {
	o := newOptions(interval, message)
	s := defaultSubmitter.Load()
	if s == nil {
		panic("repeatsubmit: 尚未初始化，请先调用 repeatsubmit.Init")
	}
	return func(c *gin.Context) {
		s.run(c, o)
	}
}
