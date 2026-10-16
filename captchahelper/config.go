package captchahelper

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/mojocn/base64Captcha"
)

// 验证码类型常量。
const (
	// CaptchaTypeMath 算术验证码，如 3+5=? 答 8。
	CaptchaTypeMath = "math"
	// CaptchaTypeChar 字符验证码，从字符池随机取 charLength 个。
	CaptchaTypeChar = "char"
	// CaptchaTypeDigit 纯数字验证码，取 numberLength 位。
	CaptchaTypeDigit = "digit"
	// CaptchaTypeChinese 中文验证码。
	CaptchaTypeChinese = "chinese"
)

// 干扰线名称，对应 base64Captcha 的 OptionShow*Line 位掩码。
const (
	LineHollow = "hollow"
	LineSlim   = "slim"
	LineSine   = "sine"
)

// lineOptions 干扰线名 → 位掩码。
var lineOptions = map[string]int{
	LineHollow: base64Captcha.OptionShowHollowLine,
	LineSlim:   base64Captcha.OptionShowSlimeLine,
	LineSine:   base64Captcha.OptionShowSineLine,
}

// embeddedFonts base64Captcha 内嵌的字体文件名。
//
// 必须是白名单：库的 LoadFontByName 读不到文件时直接 panic 而非回 error，
// 配错字体名会炸在绘图时而不是启动时，故在 Init 校验阶段先挡掉。
var embeddedFonts = map[string]bool{
	"3Dumb.ttf":             true,
	"actionj.ttf":           true,
	"ApothecaryFont.ttf":    true,
	"chromohv.ttf":          true,
	"Comismsh.ttf":          true,
	"DeborahFancyDress.ttf": true,
	"DENNEthree-dee.ttf":    true,
	"Flim-Flam.ttf":         true,
	"RitaSmith.ttf":         true,
	"wqy-microhei.ttc":      true,
}

// FontChinese 唯一带中文字形的内嵌字体，chinese 类型不配字体时默认用它。
const FontChinese = "wqy-microhei.ttc"

// DefaultCharSource 字符验证码默认字符池，去掉了易混淆的 0/O/1/l/I。
//
// 不用库的 TxtNumbers/TxtAlphabet：那两个常量自己就缺字符
// (TxtNumbers 少 5，TxtAlphabet 少 I L P W i l p w)，不如自己定一份干净的。
const DefaultCharSource = "234567890abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ"

// digit 类型的最小画布尺寸。
//
// DriverDigit 用的是点阵字模 + 随机落点，画布太小时内部
// rand.Intn 会拿到非正数直接 panic（实测 60x30 必炸）。80x40 实测安全。
const (
	minDigitWidth  = 80
	minDigitHeight = 40
)

// CaptchaConfig 验证码配置，对应 yaml 的 captcha 段。
//
// 分三层：顶层放开关/类型/长度（与 Java 版 CaptchaProperties 同名，便于对照配置），
// image 段放各类型共用的渲染参数，再按类型开专属段放只对该类型有意义的参数。
type CaptchaConfig struct {
	// Enable 是否启用验证码校验。
	Enable bool `mapstructure:"enable"`
	// Type 验证码类型：math / char / digit / chinese。
	Type string `mapstructure:"type"`
	// CharLength 字符验证码(char)的字符个数，缺省 4。
	CharLength int `mapstructure:"charLength"`
	// NumberLength 数字验证码(digit)的位数，缺省 4。
	NumberLength int `mapstructure:"numberLength"`

	// Image 图片渲染参数，math/char/chinese 共用。
	Image ImageConfig `mapstructure:"image"`
	// Char char 类型专属参数。
	Char CharConfig `mapstructure:"char"`
	// Digit digit 类型专属参数。
	Digit DigitConfig `mapstructure:"digit"`
	// Chinese chinese 类型专属参数。
	Chinese ChineseConfig `mapstructure:"chinese"`
}

// ImageConfig 图片渲染参数。
//
// 只作用于 math/char/chinese——digit 换了一套点阵渲染器，除宽高外
// 噪点/干扰线/背景色/字体它全都不读（见 base64Captcha.DriverDigit）。
type ImageConfig struct {
	// Width 图片宽度(像素)，缺省 160。
	Width int `mapstructure:"width"`
	// Height 图片高度(像素)，缺省 60。
	Height int `mapstructure:"height"`
	// NoiseCount 背景干扰字符数，缺省 0(不画)。
	NoiseCount int `mapstructure:"noiseCount"`
	// Lines 干扰线，可组合 hollow/slim/sine，缺省 ["slim"]。
	//
	// 只开细直线是刻意的：hollow 和 sine 会把字符压得不可读
	// (base64Captcha 源码自己都注了"波浪线 比较丑")，要开请先自测可读性。
	Lines []string `mapstructure:"lines"`
	// BgColor 背景色，#RGB/#RRGGBB/#RRGGBBAA，留空为随机浅色。
	BgColor string `mapstructure:"bgColor"`
	// Fonts 字体文件名，须是内嵌字体之一，留空为随机用全部字体。
	Fonts []string `mapstructure:"fonts"`
}

// CharConfig char 类型专属参数。
type CharConfig struct {
	// Source 字符池，留空用 DefaultCharSource。
	Source string `mapstructure:"source"`
}

// DigitConfig digit 类型专属参数。
type DigitConfig struct {
	// MaxSkew 单个数字的最大倾斜系数，缺省 0.7。
	MaxSkew float64 `mapstructure:"maxSkew"`
	// DotCount 背景圆点数，缺省 80。也决定调色板大小。
	DotCount int `mapstructure:"dotCount"`
}

// ChineseConfig chinese 类型专属参数。
type ChineseConfig struct {
	// Length 取几个词，缺省 4。
	Length int `mapstructure:"length"`
	// Source 中文词源，留空用库内置常用字表。
	//
	// 两种语义：不含逗号时按单字取，Length 即字数；含逗号时按词取，
	// Length 是词数而非字数，出题长度随词长浮动。
	Source string `mapstructure:"source"`
}

// 各配置项缺省值。
const (
	defaultWidth        = 160
	defaultHeight       = 60
	defaultCharLength   = 4
	defaultNumberLength = 4
	defaultCNLength     = 4
	defaultMaxSkew      = 0.7
	defaultDotCount     = 80
)

// normalize 把零值填成缺省值，返回补全后的副本。
//
// 零值一律当"未配置"处理，因此 noiseCount/lines 想显式关掉得分别写 0 和 []
// ——前者零值即关，后者 nil 会被填成 ["slim"]，要无干扰线请配 lines: []。
func (c CaptchaConfig) normalize() CaptchaConfig {
	if c.Image.Width <= 0 {
		c.Image.Width = defaultWidth
	}
	if c.Image.Height <= 0 {
		c.Image.Height = defaultHeight
	}
	if c.Image.Lines == nil {
		c.Image.Lines = []string{LineSlim}
	}
	if c.CharLength <= 0 {
		c.CharLength = defaultCharLength
	}
	if c.NumberLength <= 0 {
		c.NumberLength = defaultNumberLength
	}
	if c.Char.Source == "" {
		c.Char.Source = DefaultCharSource
	}
	if c.Digit.MaxSkew <= 0 {
		c.Digit.MaxSkew = defaultMaxSkew
	}
	if c.Digit.DotCount <= 0 {
		c.Digit.DotCount = defaultDotCount
	}
	if c.Chinese.Length <= 0 {
		c.Chinese.Length = defaultCNLength
	}
	if c.Chinese.Source == "" {
		c.Chinese.Source = base64Captcha.TxtChineseCharaters
	}
	// 中文字形只有 wqy-microhei.ttc 有，用拉丁字体画中文会出豆腐块。
	if c.Type == CaptchaTypeChinese && len(c.Image.Fonts) == 0 {
		c.Image.Fonts = []string{FontChinese}
	}
	return c
}

// validate 校验补全后的配置，错误由 Init 转成 panic。
func (c CaptchaConfig) validate() error {
	switch c.Type {
	case CaptchaTypeMath, CaptchaTypeChar, CaptchaTypeDigit, CaptchaTypeChinese:
	default:
		return fmt.Errorf("未知验证码类型 %q，可选 %s/%s/%s/%s",
			c.Type, CaptchaTypeMath, CaptchaTypeChar, CaptchaTypeDigit, CaptchaTypeChinese)
	}

	if c.Image.NoiseCount < 0 {
		return fmt.Errorf("image.noiseCount 不能为负: %d", c.Image.NoiseCount)
	}
	for _, name := range c.Image.Lines {
		if _, ok := lineOptions[name]; !ok {
			return fmt.Errorf("未知干扰线 %q，可选 %s/%s/%s", name, LineHollow, LineSlim, LineSine)
		}
	}
	if _, err := parseHexColor(c.Image.BgColor); err != nil {
		return fmt.Errorf("image.bgColor %q 不合法: %w", c.Image.BgColor, err)
	}
	for _, f := range c.Image.Fonts {
		if !embeddedFonts[f] {
			return fmt.Errorf("字体 %q 不在内嵌字体中，可选: %s", f, strings.Join(sortedFontNames(), ", "))
		}
	}

	switch c.Type {
	case CaptchaTypeChar:
		// RandText 按 rune 取，字符池长度也按 rune 算，否则中文/emoji 池会误判。
		if n := len([]rune(c.Char.Source)); n < 2 {
			return fmt.Errorf("char.source 至少需要 2 个字符，当前 %d", n)
		}
	case CaptchaTypeDigit:
		if c.Image.Width < minDigitWidth || c.Image.Height < minDigitHeight {
			return fmt.Errorf("digit 类型画布不得小于 %dx%d，当前 %dx%d（过小会导致库内部 panic）",
				minDigitWidth, minDigitHeight, c.Image.Width, c.Image.Height)
		}
	case CaptchaTypeChinese:
		// 含逗号且分段数 <= length 时，DriverChinese 会静默退化成数字+字母，
		// 压根不出中文（driver_chinese.go 的 length <= d.Length 分支）。挡在启动期。
		if segs := strings.Split(c.Chinese.Source, ","); len(segs) > 1 && len(segs) <= c.Chinese.Length {
			return fmt.Errorf("chinese.source 按逗号分了 %d 个词，须多于 chinese.length=%d，否则库会静默退化成数字+字母",
				len(segs), c.Chinese.Length)
		}
	}
	return nil
}

// parseHexColor 解析 #RGB / #RRGGBB / #RRGGBBAA，空串返回 nil(表示随机浅色)。
func parseHexColor(s string) (*color.RGBA, error) {
	if s == "" {
		return nil, nil
	}
	hex := strings.TrimPrefix(s, "#")
	if len(hex) != 3 && len(hex) != 6 && len(hex) != 8 {
		return nil, fmt.Errorf("需为 #RGB / #RRGGBB / #RRGGBBAA")
	}
	if len(hex) == 3 {
		// #abc → #aabbcc
		var sb strings.Builder
		for _, r := range hex {
			sb.WriteRune(r)
			sb.WriteRune(r)
		}
		hex = sb.String()
	}
	v, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("非十六进制")
	}
	c := &color.RGBA{A: 0xff}
	if len(hex) == 8 {
		c.R, c.G, c.B, c.A = uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v)
	} else {
		c.R, c.G, c.B = uint8(v>>16), uint8(v>>8), uint8(v)
	}
	return c, nil
}

// lineMask 把干扰线名列表折成 base64Captcha 的位掩码。
func (ic ImageConfig) lineMask() int {
	var mask int
	for _, name := range ic.Lines {
		mask |= lineOptions[name]
	}
	return mask
}

// sortedFontNames 内嵌字体名，仅用于拼错误提示。
func sortedFontNames() []string {
	names := make([]string, 0, len(embeddedFonts))
	for n := range embeddedFonts {
		names = append(names, n)
	}
	// map 遍历无序，排一下让报错信息稳定。
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names
}
