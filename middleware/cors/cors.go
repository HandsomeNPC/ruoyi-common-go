// Package cors 跨域中间件。
//
// 按 origin 通配白名单放行来源，预检(OPTIONS)回显允许的方法/头/缓存时长，
// 不合规的跨域请求直接 403。必须注册在 Auth 之前。
//
// 对应原 ruoyi-core/middleware/cors.go，抽出为独立包，不依赖 ruoyi-core/middleware。
package cors

import (
	"github.com/gin-gonic/gin"
)

// CORSWithConfig 跨域中间件。
func CORSWithConfig(cfg CORSConfig) gin.HandlerFunc {
	return handler(cfg)
}
