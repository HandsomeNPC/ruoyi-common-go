// Package accesslog 访问日志中间件。
//
// 按 content-type 分流打印入参：JSON 走 RepeatableBody 缓存的 body，表单/query 走 Form，
// 再用 defer 补一条结束日志带状态码与耗时。必须注册在 RepeatableBody 之后，否则读不到 JSON body。
//
// 参数脱敏与截断在本包自带一份（param.go），不引外部脱敏包，换取中间件包的完全自包含。
// traceId 取值与 logTracePrefix 就地复刻一份，同样不引 ruoyi-core/middleware。
package accesslog

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// AccessLogWithConfig 访问日志中间件。注册前就地铺兜底默认值并校验配置，不合规直接 panic，调用方无需手动校验。
func AccessLogWithConfig(cfg AccessLogConfig) gin.HandlerFunc {
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		panic(fmt.Sprintf("accessLog: %v", err))
	}
	return handler(cfg)
}
