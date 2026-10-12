package mailhelper

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// dialTimeout 建连超时。SMTP 服务器不可达时若不设超时，
// 请求会一直挂到客户端断开，把 handler 协程占死。
const dialTimeout = 10 * time.Second

// Mailer 零值不可用，须经 Init 构造。
type Mailer struct {
	cfg MailConfig
}

// Send 见包级 Send。
func (m *Mailer) Send(ctx context.Context, to, subject, body string) error {
	if to == "" {
		return errors.New("mailhelper: 收件人为空")
	}

	client, err := m.dial(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("mailhelper: 认证失败: %w", err)
	}
	if err := client.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("mailhelper: 设置发件人失败: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mailhelper: 设置收件人失败: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailhelper: 打开数据通道失败: %w", err)
	}
	if _, err := w.Write([]byte(m.message(to, subject, body))); err != nil {
		return fmt.Errorf("mailhelper: 写入邮件内容失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailhelper: 提交邮件失败: %w", err)
	}
	return client.Quit()
}

// dial SSL 直连（465）与 STARTTLS（587）不可混：对着 465 发 STARTTLS 会卡住。
func (m *Mailer) dial(ctx context.Context) (*smtp.Client, error) {
	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprint(m.cfg.Port))
	dialer := &net.Dialer{Timeout: dialTimeout}

	if m.cfg.SSL {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: m.cfg.Host})
		if err != nil {
			return nil, fmt.Errorf("mailhelper: 连接 %s 失败: %w", addr, err)
		}
		client, err := smtp.NewClient(conn, m.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("mailhelper: 建立 SMTP 会话失败: %w", err)
		}
		return client, nil
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("mailhelper: 连接 %s 失败: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("mailhelper: 建立 SMTP 会话失败: %w", err)
	}
	if err := client.StartTLS(&tls.Config{ServerName: m.cfg.Host}); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("mailhelper: STARTTLS 升级失败: %w", err)
	}
	return client, nil
}

// message 拼 RFC 5322 报文。
func (m *Mailer) message(to, subject, body string) string {
	var sb strings.Builder
	sb.WriteString("From: " + m.cfg.From + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	// 主题必须按 RFC 2047 编码：中文主题直接塞进报文头，多数客户端显示成乱码。
	sb.WriteString("Subject: " + encodeHeader(subject) + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	// 空行分隔头与正文，缺了整封信会被当成头解析。
	sb.WriteString("\r\n")
	sb.WriteString(normalizeNewlines(body))
	return sb.String()
}

// encodeHeader 按 RFC 2047 编码报文头里的非 ASCII 内容（ASCII 原样透传）。
func encodeHeader(s string) string {
	return mime.BEncoding.Encode("UTF-8", s)
}

// normalizeNewlines 把裸 \n 统一成 \r\n。
// SMTP 以 \r\n 为行分隔符，混用会让部分服务器把正文截断。
func normalizeNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", "\r\n")
}
