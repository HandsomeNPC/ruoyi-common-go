// Package xss 请求清洗中间件。
//
// 把请求里的 HTML 标签剔掉（保留标签内的文字），覆盖查询串、表单与 JSON body：
// JSON 走逐值清洗避免破坏结构，body 取自 RepeatableBody 缓存的 gin.BodyBytesKey。
// 必须注册在 RepeatableBody 之后、所有读参数的中间件与 handler 之前。
//
// 取舍：
//   - 自包含、不引 ruoyi-core/middleware：traceId 取值与 logTracePrefix 就地复刻一份，
//     Ant 路径匹配（ExcludeURLs 用）自带 path.go，本包零外部 middleware 依赖。
//   - GET/DELETE 默认跳过：查询串与表单多半是无害读参，清洗收益低且易误伤。
package xss

import (
	"github.com/gin-gonic/gin"
)

// XSSWithConfig 请求清洗中间件。注册前就地铺兜底默认值，调用方无需手动调 SetDefaults / Validate。
func XSSWithConfig(cfg XSSConfig) gin.HandlerFunc {
	cfg.applyDefaults()
	return handler(cfg)
}
