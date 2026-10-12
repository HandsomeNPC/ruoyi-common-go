package xss

import (
	"net/http"

	"github.com/spf13/viper"
)

// contentTypeJSON JSON 请求的 content-type 前缀。
const contentTypeJSON = "application/json"

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
