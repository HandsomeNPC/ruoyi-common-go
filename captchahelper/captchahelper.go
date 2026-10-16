// Package captchahelper 图形验证码的生成。
//
// 对照 smshelper：Init(CaptchaConfig) 构造驱动并设包级全局，业务侧用包级
// Enabled() / Generate()。本包只管出题、画图、回 uuid / 裸 base64 / 答案，
// **不碰 Redis**——把答案写进 Redis 是业务侧的事（键约定见 internal/resource/constant.CaptchaCodeKey，
// 校验在 internal/auth/service 各认证策略的 validateCaptcha 里）。
//
// 底层图形绘制用 base64Captcha 的 Driver（只用 DrawCaptcha 画图），
// 不用它的 Captcha/Store——Store 接口无 TTL 也无 ctx，塞不进业务侧的
// "global:captcha_codes: 前缀 + N 分钟过期" 约定，故答案的存取都在业务侧自管。
//
// 支持四种类型：math 算术、char 字符、digit 纯数字、chinese 中文（见 config.go）。
// 库另有 language(多语系) 与 audio(wav 语音) 两种驱动未接：前者画字硬编码只用中文字体，
// 阿拉伯/希伯来/泰语会出豆腐块；后者产出的是 audio/wav，塞不进 <img>，
// 接它得改 Generate 的签名多回一个 mimeType。
//
// 是否启用由 Enabled() 判定，响应里带不带 captchaEnabled 开关位、出不出图
// 都是业务侧（/auth/code handler）的职责，本包的 Generate 只管生成本身，
// 未就绪时回 ErrDisabled。
package captchahelper

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"

	"github.com/mojocn/base64Captcha"
)

// pngDataURIPrefix base64Captcha 的 EncodeB64string 给图片加的 data URI 前缀。
const pngDataURIPrefix = "data:" + base64Captcha.MimeTypeImage + ";base64,"

// ErrDisabled 验证码功能未开启（顶层 enable=false 或尚未 Init）。
// 由调用方翻译成给前端看的文案；出不出图由其先用 Enabled() 判定。
var ErrDisabled = errors.New("captchahelper: 验证码功能未开启")

// defaultDriver 包级默认驱动，Init 时写入，handler 读。nil 表示未启用。
// base64Captcha.Driver 是接口，atomic.Pointer 持其指针(*Driver)：Store 传 &d，
// Load 返回 nil 即未初始化/未启用，存取顺序由 atomic.Pointer 保证。
var defaultDriver atomic.Pointer[base64Captcha.Driver]

// Init 按传入的 CaptchaConfig 构造并设包级默认驱动。
// 只在启动期调一次，配置不合法或构造失败直接 panic（启动期 fail-fast）。
// !Enable 直接返回，defaultDriver 保持 nil，Enabled() 返回 false、Generate 回 ErrDisabled。
//
// 校验必须前置在此：底层库对字体名不存在、画布过小等情形是 panic 而非回 error，
// 放过去就会炸在某次用户请求的绘图里，而不是启动时。
func Init(cfg CaptchaConfig) {
	if !cfg.Enable {
		// 清空而非直接 return：Init 正常只在启动期调一次，但重复调用时
		// 若保留上一次的驱动，enable=false 就关不掉验证码了。
		defaultDriver.Store(nil)
		log.Printf("[captchahelper] 未启用")
		return
	}
	cfg = cfg.normalize()
	if err := cfg.validate(); err != nil {
		panic(fmt.Errorf("captchahelper: 配置不合法: %w", err))
	}
	d, err := newDriver(cfg)
	if err != nil {
		panic(fmt.Errorf("captchahelper: 初始化失败: %w", err))
	}
	defaultDriver.Store(&d)
	log.Printf("[captchahelper] 已就绪: type=%s size=%dx%d",
		cfg.Type, cfg.Image.Width, cfg.Image.Height)
}

// Enabled 验证码是否启用。
func Enabled() bool { return defaultDriver.Load() != nil }

// Generate 生成一条验证码，返回 uuid、base64 图片与答案。
//
// 未就绪(未 Init 或 Enable=false)时回 ErrDisabled——是否给前端出图由调用方
// 先用 Enabled() 判定，本函数只管生成本身。四类驱动统一走
// GenerateIdQuestionAnswer(出题+答案) → DrawCaptcha(画图)，uuid 复用驱动返回的
// 随机串(原项目用 simpleUUID，此处等价且少一层依赖)。图片是裸 base64（PNG，
// 不带 data URI 前缀），前缀由前端自拼。答案写 Redis 也由调用方做。
func Generate() (uuid, img, answer string, err error) {
	p := defaultDriver.Load()
	if p == nil {
		return "", "", "", ErrDisabled
	}
	d := *p
	id, question, ans := d.GenerateIdQuestionAnswer()
	item, e := d.DrawCaptcha(question)
	if e != nil {
		return "", "", "", fmt.Errorf("captchahelper: 绘制失败: %w", e)
	}
	// EncodeB64string 回带 "data:image/png;base64," 前缀，此处剥成裸 base64。
	return id, strings.TrimPrefix(item.EncodeB64string(), pngDataURIPrefix), ans, nil
}
