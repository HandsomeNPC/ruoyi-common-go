package captchahelper

// 验证码类型常量。
const (
	// CaptchaTypeMath 算术验证码，如 3 + 5 = ? 8。
	CaptchaTypeMath = "math"
	// CaptchaTypeChar 字符验证码，随机取 charLength 个字符。
	CaptchaTypeChar = "char"
)

// CaptchaConfig 验证码配置，对应 yaml 的 captcha 段。
//
// 仅暴露 3 个可配项(启用开关、类型、字符长度)；图片渲染、干扰线、字体、
// 算术题操作数范围等在 driver.go / base64Captcha 内固定写死，与原项目对齐。
type CaptchaConfig struct {
	// Enable 是否启用验证码校验。
	Enable bool `mapstructure:"enable"`
	// Type 验证码类型：math 算术计算 / char 字符验证。
	Type string `mapstructure:"type"`
	// CharLength 字符验证码的长度(字符个数)。
	CharLength int `mapstructure:"charLength"`
}
