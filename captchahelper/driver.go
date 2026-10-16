package captchahelper

import (
	"fmt"

	"github.com/mojocn/base64Captcha"
)

// newDriver 按配置构造绘图驱动。
//
// 入参须是 normalize 补全并 validate 通过的配置：本函数不再兜空值，
// 也不复查字体名/画布尺寸等（库对这些是 panic 而非回 error，校验必须前置）。
// 四类驱动都自带 GenerateIdQuestionAnswer 出题、DrawCaptcha 画图，
// 故本包不必自存类型/位数，Init 后驱动即自洽。
func newDriver(cfg CaptchaConfig) (base64Captcha.Driver, error) {
	img := cfg.Image
	bg, err := parseHexColor(img.BgColor)
	if err != nil {
		return nil, fmt.Errorf("captchahelper: 背景色不合法: %w", err)
	}

	switch cfg.Type {
	case CaptchaTypeMath:
		// 算术题的运算符与操作数范围由 DriverMath 内置(+ - x，加法 0-19/乘法 0-9/
		// 减法被减数 0-99)，库未开放配置，与原项目行为一致。
		return base64Captcha.NewDriverMath(
			img.Height, img.Width, img.NoiseCount, img.lineMask(),
			bg, nil, img.Fonts), nil

	case CaptchaTypeChar:
		return base64Captcha.NewDriverString(
			img.Height, img.Width, img.NoiseCount, img.lineMask(),
			cfg.CharLength, cfg.Char.Source, bg, nil, img.Fonts), nil

	case CaptchaTypeDigit:
		// 点阵渲染器，只认宽高 + 位数 + 倾斜 + 圆点数；
		// 噪点/干扰线/背景色/字体对它无效（库里压根不读）。
		return base64Captcha.NewDriverDigit(
			img.Height, img.Width, cfg.NumberLength,
			cfg.Digit.MaxSkew, cfg.Digit.DotCount), nil

	case CaptchaTypeChinese:
		return base64Captcha.NewDriverChinese(
			img.Height, img.Width, img.NoiseCount, img.lineMask(),
			cfg.Chinese.Length, cfg.Chinese.Source, bg, nil, img.Fonts), nil

	default:
		return nil, fmt.Errorf("captchahelper: 未知验证码类型 %q", cfg.Type)
	}
}
