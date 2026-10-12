// Package trace 链路 id 中间件。
//
// 取或生成 traceId 写进 gin.Context 与 context.Context，并回写响应头，供下游日志与调用方对齐。
// 必须注册在 CORS 之后：CORS 先写自己的头，本中间件再追加 traceId 头，避免被覆盖。
//
// 自包含、不引 ruoyi-core/middleware：与 accesslog/xss/repeatablebody 一致，日志前缀由各包就地复刻。
package trace

import (
	"github.com/gin-gonic/gin"
)

// TraceIDWithConfig 链路 id 中间件：取或生成 id，写进上下文与响应头。必须注册在 CORS 之后。
func TraceIDWithConfig(cfg TraceIDConfig) gin.HandlerFunc {
	return handler(cfg)
}
