package xss

import "net/http"

// contentTypeJSON JSON 请求的 content-type 前缀。
const contentTypeJSON = "application/json"

// defaultXSSSkipMethods XSS 默认跳过清洗的请求方法。
var defaultXSSSkipMethods = []string{http.MethodGet, http.MethodDelete}

// defaultXSSExcludeURLs XSS 默认跳过清洗的路径，Ant 风格。
var defaultXSSExcludeURLs = []string{"/system/notice", "/warm-flow/save-json"}

// XSSConfig 清洗配置。
type XSSConfig struct {
	// ExcludeURLs 跳过清洗的路径，Ant 风格。
	ExcludeURLs []string `mapstructure:"excludeUrls"`

	// SkipMethods 跳过清洗的请求方法，为空表示用 GET/DELETE。
	SkipMethods []string `mapstructure:"skipMethods"`
}

// applyDefaults 就地铺兜底默认值。ExcludeURLs / SkipMethods 为空时补默认。
func (x *XSSConfig) applyDefaults() {
	if len(x.ExcludeURLs) == 0 {
		x.ExcludeURLs = defaultXSSExcludeURLs
	}
	if len(x.SkipMethods) == 0 {
		x.SkipMethods = defaultXSSSkipMethods
	}
}
