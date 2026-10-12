package repeatablebody

import (
	"fmt"

	"github.com/spf13/viper"
)

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

// DefaultRepeatableBody 返回可重复读 body 默认配置。
func DefaultRepeatableBody() RepeatableBodyConfig {
	return RepeatableBodyConfig{
		ContentTypes: []string{contentTypeJSON},
		MaxBodySize:  defaultMaxBodySize,
	}
}

// SetDefaults 把默认值铺给 viper。
func (c RepeatableBodyConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault("repeatableBody.contentTypes", c.ContentTypes)
	v.SetDefault("repeatableBody.maxBodySize", c.MaxBodySize)
}

// Validate 校验可重复读 body 配置。
func (c RepeatableBodyConfig) Validate() error {
	if c.MaxBodySize < 0 {
		return fmt.Errorf("repeatableBody.maxBodySize 不能为负数")
	}
	return nil
}
