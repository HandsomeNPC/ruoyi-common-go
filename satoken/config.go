package satoken

// SATokenConfig sa-token-go 鉴权配置，对应 yaml 的 satoken 段。
// 默认值与校验在 Init 内统一兜底，jwtSecretKey 缺失在启动期即报错。
//
// 仅暴露 4 个可配项(token 名、并发登录、共享 token、jwt 密钥)；其余策略
// (JWT 模式、仅读 Cookie/Body 等)在 Init 里固定写死。多进程(auth / system)
// 必须读同一份值，否则 auth 签的 token 在 system 那边验不过。
type SATokenConfig struct {
	// TokenName 读取 token 的请求头名，对应 sa-token.token-name，为空视为整段未配置。
	TokenName string `mapstructure:"tokenName"`
	// IsConcurrent 是否允许同一账号并发登录(为true时允许一起登录，为false时新登录挤掉旧登录)。
	IsConcurrent bool `mapstructure:"isConcurrent"`
	// IsShare 多端登录时是否共用一个token(为true时所有登录共用一个token，为false时每次登录新建一个token)。
	IsShare bool `mapstructure:"isShare"`
	// JwtSecretKey JWT 签名密钥，必填(固定 JWT 模式)。
	JwtSecretKey string `mapstructure:"jwtSecretKey"`
}

// applyDefaults 就地铺兜底默认值。TokenName 为空视为未配置，一并补 IsConcurrent=true
// （bool 无零值可辨「未设」，故仅在整体未配置时兜底，已显式配置 TokenName 时尊重调用方的 bool）。
func (s *SATokenConfig) applyDefaults() {
	if s.TokenName == "" {
		s.TokenName = defaultTokenName
		s.IsConcurrent = true
	}
}

// validate 校验配置，Init 内调用。
func (s SATokenConfig) validate() error {
	if s.JwtSecretKey == "" {
		return errMissingSecretKey
	}
	return nil
}
