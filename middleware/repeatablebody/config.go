package repeatablebody

import "fmt"

// contentTypeJSON JSON 请求的 content-type 前缀。
const contentTypeJSON = "application/json"

// defaultMaxBodySize 默认缓存上限 10MB。
const defaultMaxBodySize = 10 << 20

// RepeatableBodyConfig 可重复读 body 的配置。
type RepeatableBodyConfig struct {
	// ContentTypes 需要缓存的 content-type 前缀（小写），大小写不敏感匹配。
	ContentTypes []string `mapstructure:"contentTypes"`

	// MaxBodySize 允许缓存的最大字节数，超出则拒绝请求。<=0 表示用默认值。
	MaxBodySize int64 `mapstructure:"maxBodySize"`
}

// applyDefaults 就地铺兜底默认值。ContentTypes 为空补 application/json；MaxBodySize 为 0 补 10MB，负数留给 validate 拦。
func (c *RepeatableBodyConfig) applyDefaults() {
	if len(c.ContentTypes) == 0 {
		c.ContentTypes = []string{contentTypeJSON}
	}
	if c.MaxBodySize == 0 {
		c.MaxBodySize = defaultMaxBodySize
	}
}

// validate 校验配置合理性，注册期调用，不合规直接 panic。
func (c RepeatableBodyConfig) validate() error {
	if c.MaxBodySize < 0 {
		return fmt.Errorf("repeatableBody.maxBodySize 不能为负数")
	}
	return nil
}
