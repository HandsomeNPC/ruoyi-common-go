// Package mailhelper 邮件发送（纯文本、单收件人）。
//
// 本项目发邮件的唯一场景是登录验证码，附件/HTML/群发都用不上，
// 引第三方库不划算，标准库 net/smtp 足够。
package mailhelper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
)

// ErrDisabled 邮件功能未开启。由调用方翻译成给前端看的文案。
var ErrDisabled = errors.New("mailhelper: 邮件功能未开启")

// 默认 SMTP 参数：yaml 未配置时 Init 兜底。
const (
	// 465 是 SMTP over SSL 的约定端口，与 SSL 直连配套。
	defaultMailPort = 465
	// 默认走 SSL 直连而非 STARTTLS：国内邮箱（QQ/163/阿里）的 465 都是
	// SSL 直连，587 的 STARTTLS 反而常被封。
	defaultMailSSL = true
)

// MailSenderHolder 写只在启动期 Init 一次、读是每条发信的热路径，故用
// atomic.Pointer 而非 RWMutex：读是无锁原子 load；nil 表示未启用。
var MailSenderHolder atomic.Pointer[Mailer]

// Init 须在配置加载后、接请求前调一次。!Enabled 直接返回；已开启但配置
// 缺失则 panic（启动期 fail-fast）。
func Init(cfg MailConfig) {
	if !cfg.Enabled {
		log.Printf("[mailhelper] 未启用")
		return
	}
	// 端口未显式配置时，SSL 一并兜默认值：bool 零值无法区分「未设」与「显式 false」，
	// 故只在端口也未设（说明用户没细配加密参数）时套 465+SSL 直连这套约定。
	// 端口已显式配置时，SSL 全听 yaml。
	if cfg.Port == 0 {
		cfg.Port = defaultMailPort
		cfg.SSL = defaultMailSSL
	}

	if err := validate(cfg); err != nil {
		panic(fmt.Errorf("mailhelper: %w", err))
	}
	m := &Mailer{cfg: cfg}
	MailSenderHolder.Store(m)
	log.Printf("[mailhelper] 已就绪: host=%s:%d ssl=%t", cfg.Host, cfg.Port, cfg.SSL)
}

func validate(cfg MailConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("host 未配置")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("port 必须是 1-65535 的端口号")
	}
	if cfg.From == "" {
		return fmt.Errorf("from 未配置")
	}
	if cfg.User == "" {
		return fmt.Errorf("user 未配置")
	}
	if cfg.Password == "" {
		return fmt.Errorf("password 未配置")
	}
	return nil
}

// Enabled 邮件功能是否开启。
func Enabled() bool { return MailSenderHolder.Load() != nil }

// ctx 只能取消建连（net.Dialer），SMTP 会话本身不可中途取消——
// 验证码邮件会话很短，不值得为此实现协议层超时。
func Send(ctx context.Context, to, subject, body string) error {
	m := MailSenderHolder.Load()
	if m == nil {
		return ErrDisabled
	}
	return m.Send(ctx, to, subject, body)
}
