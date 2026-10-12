package apiencrypt

import "fmt"

// DefaultAPIEncryptHeader 传递 AES 密钥的头名。
const DefaultAPIEncryptHeader = "encrypt-key"

// APIEncryptConfig 接口加解密配置
type APIEncryptConfig struct {
	// Enabled 是否启用。false 时中间件直通，不做任何加解密。
	Enabled bool `mapstructure:"enabled"`
	// HeaderFlag 传递 AES 密钥的请求/响应头名，为空表示用 DefaultAPIEncryptHeader。
	HeaderFlag string `mapstructure:"headerFlag"`
	// PublicKey 响应加密用的 RSA 公钥，base64 编码的 X.509 SPKI。
	PublicKey string `mapstructure:"publicKey"`
	// PrivateKey 请求解密用的 RSA 私钥，base64 编码的 PKCS#8。
	PrivateKey string `mapstructure:"privateKey"`
}

// applyDefaults 就地铺兜底默认值。HeaderFlag 为空补默认头名；其余字段无默认。
func (a *APIEncryptConfig) applyDefaults() {
	if a.HeaderFlag == "" {
		a.HeaderFlag = DefaultAPIEncryptHeader
	}
}

// validate 校验配置合理性，注册期调用，不合规直接 panic。!Enabled 时直通，不要求密钥。
func (a APIEncryptConfig) validate() error {
	if !a.Enabled {
		return nil
	}
	if a.PrivateKey == "" {
		return fmt.Errorf("apiEncrypt.privateKey 未配置")
	}
	return nil
}
