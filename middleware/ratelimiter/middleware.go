package ratelimiter

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/HandsomeNPC/ruoyi-common-go/ip"
)

// handler 生成限流中间件。默认实例在构造时取一次：Init 必须先于路由注册，
// 否则在此 panic，比拖到首次请求才崩更易定位。
func handler(o *options) gin.HandlerFunc {
	l := defaultLimiter.Load()
	if l == nil {
		panic("ratelimiter: 尚未初始化，请先调用 ratelimiter.Init")
	}
	return func(c *gin.Context) {
		l.run(c, o)
	}
}

// run 执行一次限流判定。
func (l *Limiter) run(c *gin.Context, o *options) {
	key := l.combineKey(c, o)
	remain, err := l.acquire(c.Request.Context(), key, o)
	if err != nil {
		// Redis 异常时放行：可用性优先，异常已记日志。
		log.Printf("[ratelimiter] %s %s 限流判定异常,已放行: %v",
			c.Request.Method, c.Request.URL.Path, err)
		c.Next()
		return
	}
	if remain < 0 {
		// 直接写失败响应：登记 c.Error 交给外包 Recover 渲染依赖中间件顺序，自己渲染才稳定。
		log.Printf("[ratelimiter] %s %s 触发限流, 缓存key => '%s'",
			c.Request.Method, c.Request.URL.Path, key)
		c.AbortWithStatusJSON(http.StatusOK, failBody{Code: codeFail, Msg: o.message})
		return
	}
	c.Next()
}

// combineKey 组装限流缓存键：global:rate_limit:<路径>:[<IP>:|<实例ID>:]<动态维度>。
// 动态维度为空时保留结尾冒号。
func (l *Limiter) combineKey(c *gin.Context, o *options) string {
	var sb strings.Builder
	sb.WriteString(rateLimitKey)
	sb.WriteString(c.Request.URL.Path)
	sb.WriteByte(':')

	switch o.limitType {
	case LimitTypeIP:
		sb.WriteString(ip.ResolveClientIP(c.Request))
		sb.WriteByte(':')
	case LimitTypeCluster:
		sb.WriteString(l.instanceID)
		sb.WriteByte(':')
	}

	if o.keyFunc != nil {
		sb.WriteString(o.keyFunc(c))
	}
	return sb.String()
}
