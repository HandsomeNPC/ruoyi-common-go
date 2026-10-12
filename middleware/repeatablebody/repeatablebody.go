// Package repeatablebody 可重复读请求体中间件。
//
// 对配置的 content-type 把请求体读入内存并塞回 gin.BodyBytesKey，让后续中间件
// （XSS 清洗、访问日志、加密解密、防重提交指纹）与 handler 都能重复读 body。
// 超出 MaxBodySize 的请求直接以统一响应拒绝。
//
// 下游取已缓存的请求体直接用 gin 上下文，无需本包：
//
//	v, _ := c.Get(gin.BodyBytesKey)
//	body, _ := v.([]byte)
//
// 取舍：
//   - 自包含、不引入 errs：超限走 response.Fail 直渲，读取异常登记原始 error 由 Recover 兜底。
//   - 仅缓存声明的 content-type，避免对 multipart/二进制无谓占内存。
package repeatablebody

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// RepeatableBodyWithConfig 可重复读请求体中间件。注册前就地铺兜底默认值并校验配置，不合规直接 panic，调用方无需手动校验。
func RepeatableBodyWithConfig(cfg RepeatableBodyConfig) gin.HandlerFunc {
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		panic(fmt.Sprintf("repeatableBody: %v", err))
	}
	return handler(cfg)
}
