package oauth2

import "fmt"

// ProviderConfig 单个三方平台的配置，对应 yaml 的 social.type.<source>。
//
// 字段只保留已接入平台用得上的五项。Java SocialLoginConfigProperties 还有
// unionId/tenantId/agentId/alipayPublicKey 等未接入平台专属参数，接入时再补。
//
// mapstructure 标签是给调用方的 viper 用的；本包不 import viper，
// 默认值铺不铺都不影响解析——整段留空时 Type 为 nil map，按「无可用平台」处理。
type ProviderConfig struct {
	// ClientID 应用 ID。
	ClientID string `mapstructure:"clientId"`
	// ClientSecret 应用密钥。
	ClientSecret string `mapstructure:"clientSecret"`
	// RedirectURI 回调地址，指向前端回调页，须与平台控制台登记的完全一致。
	RedirectURI string `mapstructure:"redirectUri"`
	// ServerURL 自托管授权服务器地址，仅私有化部署平台(maxkey/topiam)需要。
	ServerURL string `mapstructure:"serverUrl"`
	// Scopes 请求的权限范围，留空则用各平台的默认值。
	Scopes []string `mapstructure:"scopes"`
}

// configured 是否配了基本凭据。两项缺一即视为未启用。
func (c ProviderConfig) configured() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

// Config 三方登录配置，对应 yaml 的 social 段。
type Config struct {
	// Type 按平台标识(gitee/github/maxkey/topiam)索引的配置表。
	Type map[string]ProviderConfig `mapstructure:"type"`
}

// Validate 校验配置，供调用方在启动期调用。
//
// 只校验「填了一半」的平台：整项留空表示不启用，跳过即可；
// 但配了 clientId 却漏了 secret/回调地址，是会在运行期才暴露的配置事故。
func (c Config) Validate() error {
	for source, pc := range c.Type {
		if pc.ClientID == "" && pc.ClientSecret == "" {
			continue
		}
		if pc.ClientID == "" {
			return fmt.Errorf("social.type.%s.clientId 未配置", source)
		}
		if pc.ClientSecret == "" {
			return fmt.Errorf("social.type.%s.clientSecret 未配置", source)
		}
		if pc.RedirectURI == "" {
			return fmt.Errorf("social.type.%s.redirectUri 未配置", source)
		}
	}
	return nil
}
