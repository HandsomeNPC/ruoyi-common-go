package middleware

import (
	"net/http"

	"github.com/spf13/viper"
)

// TraceIDHeader 读写链路 id 的请求/响应头名。
const TraceIDHeader = "X-Request-Id"

// defaultXSSSkipMethods XSS 默认跳过清洗的请求方法。
var defaultXSSSkipMethods = []string{http.MethodGet, http.MethodDelete}

// XSSConfig 清洗配置。
type XSSConfig struct {
	// ExcludeURLs 跳过清洗的路径，Ant 风格。
	ExcludeURLs []string `mapstructure:"excludeUrls"`

	// SkipMethods 跳过清洗的请求方法，为空表示用 GET/DELETE。
	SkipMethods []string `mapstructure:"skipMethods"`
}

// DefaultXSS 返回清洗默认配置。
func DefaultXSS() XSSConfig {
	return XSSConfig{
		ExcludeURLs: []string{"/system/notice", "/warm-flow/save-json"},
		SkipMethods: defaultXSSSkipMethods,
	}
}

// SetDefaults 把默认值铺给 viper。
func (x XSSConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault("xss.excludeUrls", x.ExcludeURLs)
	v.SetDefault("xss.skipMethods", x.SkipMethods)
}

// Validate 校验 XSS 配置。
func (x XSSConfig) Validate() error { return nil }

// TraceIDConfig 链路 id 中间件配置。
type TraceIDConfig struct {
	// Header 读写链路 id 的头名，默认 TraceIDHeader。
	Header string `mapstructure:"header"`

	// TrustInbound 是否沿用入站请求头里已有的 id。
	TrustInbound bool `mapstructure:"trustInbound"`
}

// DefaultTraceID 返回链路 id 默认配置。
func DefaultTraceID() TraceIDConfig {
	return TraceIDConfig{
		Header:       TraceIDHeader,
		TrustInbound: true,
	}
}

// SetDefaults 把默认值铺给 viper。
func (t TraceIDConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault("traceId.header", t.Header)
	v.SetDefault("traceId.trustInbound", t.TrustInbound)
}

// Validate 校验链路 id 配置。
func (t TraceIDConfig) Validate() error { return nil }
