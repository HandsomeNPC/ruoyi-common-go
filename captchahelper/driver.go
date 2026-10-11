package captchahelper

import (
	"fmt"

	"github.com/mojocn/base64Captcha"
)

// 图片渲染参数。
// 按 CaptchaConfig 的设计固定写死，不做配置项。
const (
	imgWidth  = 160 // 图片宽度(像素)
	imgHeight = 60  // 图片高度(像素)

	// noiseCount 背景干扰字符数。
	noiseCount = 0
	// showLineOptions 干扰线：只开细直线。
	// 库里另两种(空心线/正弦线)会把字符压得看不清——base64Captcha 源码自己都注了
	// "波浪线 比较丑"，实测 hollow|sine 组合基本不可读，故只保留细直线。
	showLineOptions = base64Captcha.OptionShowSlimeLine
)

// charSource 字符验证码取值集合，去掉了易混淆的 0/O/1/l/I。
const charSource = "234567890abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ"

// newDriver 按类型构造绘图驱动。
// math/char 两类驱动都自带 GenerateIdQuestionAnswer 出题、DrawCaptcha 画图，
// 本包不再自存类型/位数，Init 后驱动即自洽。
func newDriver(typ string, charLength int) (base64Captcha.Driver, error) {
	switch typ {
	case CaptchaTypeMath:
		// 算术题(操作数范围、运算符)由 DriverMath.GenerateIdQuestionAnswer 内置，
		// 与原项目对齐，本包不再自行生成。
		return base64Captcha.NewDriverMath(
			imgHeight, imgWidth, noiseCount, showLineOptions, nil, nil, nil), nil
	case CaptchaTypeChar:
		return base64Captcha.NewDriverString(
			imgHeight, imgWidth, noiseCount, showLineOptions,
			charLength, charSource, nil, nil, nil), nil
	default:
		return nil, fmt.Errorf("captchahelper: 未知验证码类型 %q", typ)
	}
}
