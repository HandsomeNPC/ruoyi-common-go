package cors

import (
	"fmt"
	"time"
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

// applyDefaults 就地铺兜底默认值。全零配置（yaml 未写 cors 段）直接铺整份默认；
// 否则只补空字段，尊重调用方显式设置的 bool。
func (c *CORSConfig) applyDefaults() {
	if len(c.AllowedOriginPatterns) == 0 && len(c.AllowedMethods) == 0 && len(c.AllowedHeaders) == 0 {
		*c = defaultCORSConfig()
		return
	}
	if len(c.AllowedOriginPatterns) == 0 {
		c.AllowedOriginPatterns = []string{"*"}
	}
	if len(c.AllowedMethods) == 0 {
		c.AllowedMethods = []string{"*"}
	}
	if len(c.AllowedHeaders) == 0 {
		c.AllowedHeaders = []string{"*"}
	}
	if len(c.ExposedHeaders) == 0 {
		c.ExposedHeaders = []string{traceIDHeader, "Content-Disposition", "download-filename"}
	}
	if c.MaxAgeSeconds == 0 {
		c.MaxAgeSeconds = defaultCORSMaxAgeSeconds
	}
}

// validate 校验配置合理性，注册期调用，不合规直接 panic。
func (c CORSConfig) validate() error {
	if c.MaxAgeSeconds < 0 {
		return fmt.Errorf("cors.maxAgeSeconds 不能为负数")
	}
	if len(c.AllowedOriginPatterns) == 0 {
		return fmt.Errorf("cors.allowedOriginPatterns 未配置")
	}
	return nil
}

// defaultCORSConfig 返回整份兜底默认配置。
func defaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowCredentials:      true,
		AllowedOriginPatterns: []string{"*"},
		AllowedMethods:        []string{"*"},
		AllowedHeaders:        []string{"*"},
		ExposedHeaders:        []string{traceIDHeader, "Content-Disposition", "download-filename"},
		MaxAgeSeconds:         defaultCORSMaxAgeSeconds,
	}
}
