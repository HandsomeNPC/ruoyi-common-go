// Package cors 跨域中间件。
//
// 按 origin 通配白名单放行来源，预检(OPTIONS)回显允许的方法/头/缓存时长，
// 不合规的跨域请求直接 403。必须注册在 Auth 之前。
//
// 对应原 ruoyi-core/middleware/cors.go，抽出为独立包，不依赖 ruoyi-core/middleware。
package cors

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// CORSWithConfig 跨域中间件。注册前就地铺兜底默认值并校验配置，不合规直接 panic，调用方无需手动校验。
func CORSWithConfig(cfg CORSConfig) gin.HandlerFunc {
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		panic(fmt.Sprintf("cors: %v", err))
	}
	return handler(cfg)
}
