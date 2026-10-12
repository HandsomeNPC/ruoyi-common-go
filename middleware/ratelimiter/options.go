package ratelimiter

import (
	"time"

	"github.com/gin-gonic/gin"
)

// LimitType 限流类型。
type LimitType int

const (
	LimitTypeGlobal  LimitType = iota // 全局限流（所有调用方共用一个额度）
	LimitTypeIP                       // 按请求方 IP 限流
	LimitTypeCluster                  // 按实例限流（多实例各自独立额度）
)

const (
	defaultTime    = 60 * time.Second // 默认限流窗口
	defaultCount   = 100              // 默认窗口内允许的次数
	defaultTimeout = 24 * time.Hour   // 键存活时间，实际取 max(window, timeout)
)

const defaultMessage = "访问过于频繁，请稍候再试" // 默认提示文案

type options struct {
	window    time.Duration             // 限流窗口
	count     int                       // 窗口内允许的次数
	limitType LimitType                 // 限流类型
	timeout   time.Duration             // 键存活时间
	keyFunc   func(*gin.Context) string // 动态维度取值，nil 表示不加
	message   string                    // 触发限流时的提示文案
}

// newOptions 构造限流配置。window/count/timeout 为零值、message 为空时取默认值。
func newOptions(window time.Duration, count int, limitType LimitType, timeout time.Duration, message string) *options {
	if window == 0 {
		window = defaultTime
	}
	if count == 0 {
		count = defaultCount
	}
	if timeout == 0 {
		timeout = defaultTimeout
	}
	if message == "" {
		message = defaultMessage
	}
	return &options{window: window, count: count, limitType: limitType, timeout: timeout, message: message}
}
