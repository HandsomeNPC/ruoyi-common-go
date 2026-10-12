package accesslog

import (
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// handler 生成访问日志中间件，必须注册在 RepeatableBody 之后。
func handler(cfg AccessLogConfig) gin.HandlerFunc {
	maxLen := cfg.MaxParamLength
	if maxLen <= 0 {
		maxLen = defaultMaxParamLength
	}
	skip := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skip[p] = struct{}{}
	}

	return func(c *gin.Context) {
		// 用 URL.Path 而非 Request.RequestURI：后者带原始查询串，会把 ?password=xxx 原样写进日志。
		path := c.Request.URL.Path
		if _, ok := skip[path]; ok {
			c.Next()
			return
		}

		logRequestStart(c, path, maxLen)

		start := time.Now()
		// 用 defer 而非 c.Next() 之后直接打，保证开始必有结束。
		defer func() {
			log.Printf("[access]%s 结束请求 => URL[%s %s],状态[%d],耗时:[%d]毫秒",
				logTracePrefix(c), c.Request.Method, path,
				c.Writer.Status(), time.Since(start).Milliseconds())
		}()

		c.Next()
	}
}

// logRequestStart 打印入参日志。
func logRequestStart(c *gin.Context, path string, maxLen int) {
	prefix, method := logTracePrefix(c), c.Request.Method

	// 按 content-type 分流，不看请求方法。
	if isJSONRequest(c) {
		v, _ := c.Get(gin.BodyBytesKey)
		body, _ := v.([]byte)
		log.Printf("[access]%s 开始请求 => URL[%s %s],参数类型[json],参数:[%s]",
			prefix, method, path, sanitizeJSONParam(body, maxLen, nil))
		return
	}

	params := sanitizeFormParam(c, maxLen, nil)
	if params == "" {
		log.Printf("[access]%s 开始请求 => URL[%s %s],无参数", prefix, method, path)
		return
	}
	log.Printf("[access]%s 开始请求 => URL[%s %s],参数类型[param],参数:[%s]",
		prefix, method, path, params)
}

// isJSONRequest 判断是否 JSON 请求。前缀匹配且大小写不敏感。
func isJSONRequest(c *gin.Context) bool {
	return strings.HasPrefix(strings.ToLower(c.ContentType()), contentTypeJSON)
}

// traceIDKey 与 ruoyi-core/middleware.TraceIDKey 同值。logTracePrefix 在 ruoyi-core/middleware
// 中未导出，这里就地复刻一份，不为本包单独引一个 middleware 依赖。
const traceIDKey = "traceId"

// logTracePrefix 返回日志用的 " [traceId]" 片段，无 id 时返回空串。
func logTracePrefix(c *gin.Context) string {
	if id := c.GetString(traceIDKey); id != "" {
		return " [" + id + "]"
	}
	return ""
}
