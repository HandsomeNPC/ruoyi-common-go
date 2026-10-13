package operlog

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/constant"
	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/enum"
	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/errs"
	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/model/response"
	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/utils/loginhelper"

	"github.com/HandsomeNPC/ruoyi-common-go/jsonx"

	"github.com/HandsomeNPC/ruoyi-common-go/ip"
)

// run 执行一次操作日志记录。
func run(c *gin.Context, title string, businessType enum.BusinessType, o *options) {
	evt := &Event{
		Title:         title,
		BusinessType:  businessType.Int(),
		OperatorType:  o.operatorType.Int(),
		Method:        handlerName(c),
		RequestMethod: c.Request.Method,
		OperURL:       limit(c.Request.URL.Path, maxURLLength),
		// X-Forwarded-For 可伪造且不校验格式，长度必须自己兜。
		OperIP:    limit(ip.ResolveClientIP(c.Request), maxIPLength),
		ClientKey: limit(c.GetHeader(constant.ClientIDHeader), maxClientKeyLength),
		Status:    enum.BusinessStatusSuccess.Int(),
	}
	fillLoginUser(c, evt)

	// 入参在 handler 之前采集：body 是一次性流，handler 绑定后再读就没了。
	if o.saveRequestData {
		evt.OperParam = requestParam(c, o.excludeParams)
	}

	// 边写边抄：不缓冲后补发，避免 panic 时抄本与真实响应不一致。
	original := c.Writer
	capture := &responseCapture{ResponseWriter: original}
	if o.saveResponseData {
		c.Writer = capture
	}

	start := time.Now()
	completed := false
	defer func() {
		c.Writer = original
		evt.CostTime = time.Since(start).Milliseconds()

		if !completed {
			// panic 正在展开：记失败后继续往上抛，不 recover。
			evt.Status = enum.BusinessStatusFail.Int()
			evt.ErrorMsg = msgPanic
			emit(c, evt)
			return
		}

		// 响应体既落 json_result，也是 failed 判定业务 code 的依据。
		// 未开启响应采集时为空串，failed 退化为只看 c.Errors 与 HTTP 状态。
		body := capture.text()
		if o.saveResponseData {
			evt.JSONResult = limit(body, maxContentLength)
		}
		if fail, reason := failed(c, body); fail {
			evt.Status = enum.BusinessStatusFail.Int()
			evt.ErrorMsg = limit(reason, maxContentLength)
		}
		emit(c, evt)
	}()

	c.Next()
	completed = true
}

// emit 把事件交给 Recorder。ctx 用 WithoutCancel：异步落库，请求 ctx 随响应取消，不脱开会落库失败。
func emit(c *gin.Context, evt *Event) {
	r := get()
	if r == nil {
		log.Printf("[operlog] 未注册 Recorder,操作日志丢弃: %s %s", evt.RequestMethod, evt.OperURL)
		return
	}
	r(context.WithoutCancel(c.Request.Context()), evt)
}

// fillLoginUser 填充登录用户字段，未登录时留零。逐字段截断：会话与请求头内容不受本进程控制，
// 超长会让整条日志落库失败——那时只剩一行错误日志，操作记录静默丢失。
func fillLoginUser(c *gin.Context, evt *Event) {
	lu := loginhelper.GetLoginUser(c)
	if lu == nil {
		return
	}
	evt.OperName = limit(lu.Username, maxNameLength)
	evt.UserID = lu.UserID
	evt.DeptID = lu.DeptID
	evt.DeptName = limit(lu.DeptName, maxNameLength)
	evt.DeviceType = limit(lu.DeviceType, maxClientKeyLength)
	evt.Browser = limit(lu.Browser, maxNameLength)
	evt.OS = limit(lu.OS, maxNameLength)
	// 请求头没带 clientid 时回落会话里的。
	if evt.ClientKey == "" {
		evt.ClientKey = limit(lu.ClientKey, maxClientKeyLength)
	}
}

// failed 判定本次请求是否失败，与 repeatsubmit 用同一组信号：c.Error 非空、HTTP 非 2xx、
// 响应 code 非 200。只看 c.Errors 不够：handler 可能直接 c.JSON(200, Fail(...)) 不登记错误，
// 会记出 status=成功却 code:500 的矛盾日志；与 repeatsubmit 口径不一致还会让同次请求在两个
// 中间件里得出相反结论。body 非 R 结构（文件流等）时按成功处理。返回 reason 供 ErrorMsg 使用。
func failed(c *gin.Context, body string) (bool, string) {
	if err := lastError(c); err != nil {
		return true, errorMsg(err)
	}
	if status := c.Writer.Status(); status < 200 || status > 299 {
		return true, fmt.Sprintf("HTTP %d", status)
	}
	code, msg, ok := parseResult(body)
	if !ok || code == response.CodeSuccess {
		return false, ""
	}
	if msg == "" {
		msg = fmt.Sprintf("业务码 %d", code)
	}
	return true, msg
}

// lastError 取 handler 登记的最后一个错误。本项目 handler 走 c.Error + Recover 渲染，
// 故业务失败与异常在此等价。
func lastError(c *gin.Context) error {
	if len(c.Errors) == 0 {
		return nil
	}
	return c.Errors.Last().Err
}

// parseResult 从响应体取业务 code 与 msg，ok=false 表示不是 R 结构。
func parseResult(body string) (int, string, bool) {
	if body == "" {
		return 0, "", false
	}
	var r struct {
		Code *int   `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := jsonx.Unmarshal([]byte(body), &r); err != nil || r.Code == nil {
		return 0, "", false
	}
	return *r.Code, r.Msg, true
}

// errorMsg 取错误文案。ServiceError 的 Detail 只进服务端日志，操作日志给管理员看，故拼上便于定位。
func errorMsg(err error) string {
	var se *errs.ServiceError
	if errors.As(err, &se) && se.Detail != "" {
		return se.Msg + " | " + se.Detail
	}
	return err.Error()
}

// handlerName 取 handler 函数名作为 Method，补 "()"。HandlerName 形如 ...(*ClientApi).Add-fm，去掉 -fm 后缀。
func handlerName(c *gin.Context) string {
	name := strings.TrimSuffix(c.HandlerName(), "-fm")
	if name == "" {
		return ""
	}
	return limit(name+"()", maxContentLength)
}

// truncatedSuffix 截断标记。
const truncatedSuffix = "...(已截断)"

// limit 按字符数截断，超长时截到 maxLen 内并追加标记，输出不超过 maxLen 个字符
// （列宽兜底：小列如 oper_ip varchar(128) 加标记后仍不得超宽）。按 rune 而非 byte
// 截：中文按字节切会把最后一个字切成乱码。
func limit(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s // 字节数没超，rune 数必没超
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	cut := maxLen - utf8.RuneCountInString(truncatedSuffix)
	if cut <= 0 {
		return sliceRunes(s, maxLen) // 列宽放不下标记，静默截断
	}
	return sliceRunes(s, cut) + truncatedSuffix
}

// sliceRunes 取 s 前 n 个 rune 的切片，n 超过 rune 数时返回原串。
func sliceRunes(s string, n int) string {
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}
