// Package smshelper 阿里云短信发送。
//
// 仅接阿里云一家：本项目发短信的唯一场景是登录验证码，sms4j 那套多渠道聚合
// 的复杂度不值得照搬。签名 / 模板随业务变，由调用方各自透传，客户端只建一个。
// 阿里云 SDK 相关代码集中在 ali.go，本文件只放与厂商无关的包级状态与入口。
package smshelper

import (
	"errors"
	"log"
)

// ErrDisabled 短信功能未开启（顶层 enabled=false，或指定渠道客户端未就绪）。
// 由调用方翻译成给前端看的文案。
var ErrDisabled = errors.New("smshelper: 短信功能未开启")

// enabled 是否有至少一个渠道就绪可发。
var enabled bool

// Init 按传入的 SMSConfig 初始化短信渠道，须在配置加载之后、开始接请求之前调用。
//
// 只在启动期调一次：写 enabled / aliClient 在 main goroutine 完成，后续 handler
// goroutine 由 net/http 创建时即与主 goroutine 建立 happens-before，故读无需加锁。
// 某家渠道失败只记日志跳过，不让一个坏渠道拖垮整组；全失败则 enabled=false，
// 接口返回「未开启」而非 fatal。
func Init(c SMSConfig) {
	if !c.Enabled {
		log.Printf("[smshelper] 未开启")
		return
	}

	if err := initAliClient(c.Aliyun); err != nil {
		log.Printf("[smshelper] 阿里云渠道未就绪: %v", err)
	} else {
		enabled = true
	}

	if enabled {
		log.Printf("[smshelper] 已就绪")
	} else {
		log.Printf("[smshelper] 无可用渠道")
	}
}

// Enabled 短信功能是否可用（至少一个渠道就绪）。
func Enabled() bool { return enabled }
