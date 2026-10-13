package repeatablebody

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// handler 生成可重复读 body 中间件。
//
// 超限直接回 413：既超限则后续链路不再执行，直接 Abort 即可，不引入 response/errs。
func handler(cfg RepeatableBodyConfig) gin.HandlerFunc {
	maxSize := cfg.MaxBodySize
	if maxSize <= 0 {
		maxSize = defaultMaxBodySize
	}
	types := make([]string, 0, len(cfg.ContentTypes))
	for _, t := range cfg.ContentTypes {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			types = append(types, t)
		}
	}

	return func(c *gin.Context) {
		if !shouldBufferBody(c, types) {
			c.Next()
			return
		}

		if c.Request.ContentLength > maxSize {
			rejectOversizedBody(c, c.Request.ContentLength, maxSize)
			return
		}

		// 多读 1 字节用来区分「刚好到上限」和「已经超了」。
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxSize+1))
		if err != nil {
			// 读取失败是系统异常，登记原始 err 交由 Recover 兜底（带错误编号），不在此直渲。
			log.Printf("[body]%s 请求地址'%s',读取请求体失败: %v",
				logTracePrefix(c), c.Request.URL.Path, err)
			_ = c.Error(err)
			c.Abort()
			return
		}
		if int64(len(body)) > maxSize {
			rejectOversizedBody(c, int64(len(body)), maxSize)
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Set(gin.BodyBytesKey, body)

		c.Next()
	}
}

// shouldBufferBody 判断本次请求是否需要缓存 body。
func shouldBufferBody(c *gin.Context, types []string) bool {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return false
	}

	ct := strings.ToLower(c.ContentType())
	for _, t := range types {
		if strings.HasPrefix(ct, t) {
			return true
		}
	}
	return false
}

// rejectOversizedBody 拒绝超限的请求体，直接回写 413。
func rejectOversizedBody(c *gin.Context, size, maxSize int64) {
	log.Printf("[body]%s 请求地址'%s',请求体超限: %d > %d 字节",
		logTracePrefix(c), c.Request.URL.Path, size, maxSize)
	c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, "请求体超出大小限制")
}

// traceIDKey 与 ruoyi-core/middleware.TraceIDKey 同值，这里直接用字面量避免反向依赖 middleware 包。
const traceIDKey = "traceId"

// logTracePrefix 返回日志用的 " [traceId]" 片段，无 id 时返回空串。
func logTracePrefix(c *gin.Context) string {
	if id := c.GetString(traceIDKey); id != "" {
		return " [" + id + "]"
	}
	return ""
}
