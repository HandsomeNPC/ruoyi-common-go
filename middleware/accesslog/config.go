package accesslog

import "fmt"

// defaultMaxParamLength 参数日志的最大长度（字符数）。
const defaultMaxParamLength = 4000

// AccessLogConfig 请求日志配置。
type AccessLogConfig struct {
	// MaxParamLength 参数日志最大字符数，<=0 表示用默认值 4000。
	MaxParamLength int `mapstructure:"maxParamLength"`

	// SkipPaths 不打日志的路径（精确匹配 URL.Path）。
	SkipPaths []string `mapstructure:"skipPaths"`
}

// applyDefaults 就地铺兜底默认值。MaxParamLength 为 0 视为未配置，补 4000；负数留给 validate 拦。
func (a *AccessLogConfig) applyDefaults() {
	if a.MaxParamLength == 0 {
		a.MaxParamLength = defaultMaxParamLength
	}
}

// validate 校验配置合理性，注册期调用，不合规直接 panic。
func (a AccessLogConfig) validate() error {
	if a.MaxParamLength < 0 {
		return fmt.Errorf("accessLog.maxParamLength 不能为负数")
	}
	return nil
}
