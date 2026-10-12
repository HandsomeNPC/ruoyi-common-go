package cors

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// defaultCORSMaxAgeSeconds 预检结果缓存时长。
const defaultCORSMaxAgeSeconds = 1800

// traceIDHeader 链路 id 响应头名，与 ruoyi-core/middleware.TraceIDHeader 同值。
// 就地复刻一份换本包自包含（不引 ruoyi-core/middleware）；trace 中间件另行迁移后会与此处重复，届时再合并。
const traceIDHeader = "X-Request-Id"

// CORSConfig 跨域配置。
type CORSConfig struct {
	// AllowCredentials 是否允许携带凭证(cookie / Authorization)。
	AllowCredentials bool `mapstructure:"allowCredentials"`

	// AllowedOriginPatterns 允许的来源，支持 * 通配，如 https://*.example.com。
	AllowedOriginPatterns []string `mapstructure:"allowedOriginPatterns"`

	// AllowedMethods 允许的请求方法，["*"] 表示回显预检请求的方法。
	AllowedMethods []string `mapstructure:"allowedMethods"`

	// AllowedHeaders 允许的请求头，["*"] 表示回显预检请求的头。
	AllowedHeaders []string `mapstructure:"allowedHeaders"`

	// ExposedHeaders 允许前端 JS 读取的响应头。
	ExposedHeaders []string `mapstructure:"exposedHeaders"`

	// MaxAgeSeconds 预检结果缓存时长(秒)，取值用 MaxAge() 方法。
	MaxAgeSeconds int `mapstructure:"maxAgeSeconds"`
}

// MaxAge 返回预检结果缓存时长。
func (c CORSConfig) MaxAge() time.Duration {
	return time.Duration(c.MaxAgeSeconds) * time.Second
}

// DefaultCORS 返回跨域默认配置。
func DefaultCORS() CORSConfig {
	return CORSConfig{
		AllowCredentials:      true,
		AllowedOriginPatterns: []string{"*"},
		AllowedMethods:        []string{"*"},
		AllowedHeaders:        []string{"*"},
		ExposedHeaders:        []string{traceIDHeader, "Content-Disposition", "download-filename"},
		MaxAgeSeconds:         defaultCORSMaxAgeSeconds,
	}
}

// SetDefaults 把默认值铺给 viper。
func (c CORSConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault("cors.allowCredentials", c.AllowCredentials)
	v.SetDefault("cors.allowedOriginPatterns", c.AllowedOriginPatterns)
	v.SetDefault("cors.allowedMethods", c.AllowedMethods)
	v.SetDefault("cors.allowedHeaders", c.AllowedHeaders)
	v.SetDefault("cors.exposedHeaders", c.ExposedHeaders)
	v.SetDefault("cors.maxAgeSeconds", c.MaxAgeSeconds)
}

// Validate 校验跨域配置。
func (c CORSConfig) Validate() error {
	if c.MaxAgeSeconds < 0 {
		return fmt.Errorf("cors.maxAgeSeconds 不能为负数")
	}
	if len(c.AllowedOriginPatterns) == 0 {
		return fmt.Errorf("cors.allowedOriginPatterns 未配置")
	}
	return nil
}
