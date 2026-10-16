package satoken

import "fmt"

// SATokenConfig sa-token-go 鉴权配置，对应 yaml 的 satoken 段。
// 默认值与校验在 Init 内统一兜底，jwtSecretKey 缺失在启动期即报错。
//
// 只暴露确有必要配置的项；其余策略(JWT 模式、读 Cookie/Body 等)在 Init 里固定写死。
// 多进程(auth / system)必须读同一份值，否则 auth 签的 token 在 system 那边验不过。
type SATokenConfig struct {
	// TokenName 读取 token 的请求头名，对应 sa-token.token-name，为空视为整段未配置。
	TokenName string `mapstructure:"tokenName"`
	// IsConcurrent 是否允许同一账号并发登录(为true时允许一起登录，为false时新登录挤掉旧登录)。
	IsConcurrent bool `mapstructure:"isConcurrent"`
	// IsShare 多端登录时是否共用一个token(为true时所有登录共用一个token，为false时每次登录新建一个token)。
	IsShare bool `mapstructure:"isShare"`
	// JwtSecretKey JWT 签名密钥，必填(固定 JWT 模式)。
	JwtSecretKey string `mapstructure:"jwtSecretKey"`

	// Timeout token 固定有效期(秒)，-1 永不过期，0 视为未配置取 defaultTimeout。
	//
	// 这是绝对上限：到点即失效，不会因用户活跃而自动延长。
	Timeout int64 `mapstructure:"timeout"`
	// ActiveTimeout token 最低活跃频率(秒)，-1 不限制(永不冻结)，0 视为未配置取 defaultActiveTimeout。
	//
	// 滑动空闲窗口：超过此时长无请求即冻结，每次鉴权通过会重新计时。
	ActiveTimeout int64 `mapstructure:"activeTimeout"`
	// DynamicActiveTimeout 是否允许按客户端覆盖上面两个超时。
	//
	// RuoYi 的 sys_client 表按客户端配 timeout / active_timeout，需要本项为 true 才生效。
	DynamicActiveTimeout bool `mapstructure:"dynamicActiveTimeout"`
}

// applyDefaults 就地铺兜底默认值。TokenName 为空视为未配置，一并补 IsConcurrent=true
// （bool 无零值可辨「未设」，故仅在整体未配置时兜底，已显式配置 TokenName 时尊重调用方的 bool）。
func (s *SATokenConfig) applyDefaults() {
	if s.TokenName == "" {
		s.TokenName = defaultTokenName
		s.IsConcurrent = true
	}
	// 0 不可能是有意的超时值（那等于登录即失效），一律视为未配置
	if s.Timeout == 0 {
		s.Timeout = defaultTimeout
	}
	if s.ActiveTimeout == 0 {
		s.ActiveTimeout = defaultActiveTimeout
	}
}

// validate 校验配置，Init 内调用。
func (s SATokenConfig) validate() error {
	if s.JwtSecretKey == "" {
		return errMissingSecretKey
	}
	if s.Timeout < -1 {
		return fmt.Errorf("satoken.timeout 非法: %d（应为 -1 或正数）", s.Timeout)
	}
	if s.ActiveTimeout < -1 {
		return fmt.Errorf("satoken.activeTimeout 非法: %d（应为 -1 或正数）", s.ActiveTimeout)
	}
	// 活跃超时不该大于固定有效期：那样它永远先被固定有效期截断，等同没配
	if s.Timeout > 0 && s.ActiveTimeout > s.Timeout {
		return fmt.Errorf("satoken.activeTimeout(%d) 不应大于 timeout(%d)", s.ActiveTimeout, s.Timeout)
	}
	return nil
}
