// Package ratelimiter 接口限流注解（装饰器）。
//
// 取舍：
//   - 滑动窗口用自写 Lua(ZSET) 单次 EVAL 原子完成（无 Redisson 等价库）。
//   - 动态 key 用闭包 RateLimiterWithKeyFunc 替代 SpEL（无 SpEL，编译期安全、无反射）。
//   - 每次按当前参数计算，count/time 改动立即生效（不复刻 hsetnx 缓存配置的坑）。
package ratelimiter

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

// codeFail 业务失败码，与 ruoyi-core/model/response.CodeFail 同值；限流触发时回此码。
// 就地定义避免引 ruoyi-core/model/response 包。
const codeFail = 500

// rateLimitKey 限流缓存键前缀。"global:" 前缀沿用全局 redis key 约定。
const rateLimitKey = "global:rate_limit:"

// Limiter 限流器。
type Limiter struct {
	instanceID string          // 本进程实例标识，仅 LimitTypeCluster 用到
	rdb        *goredis.Client // 限流计数所用的 Redis 客户端，由 Init 注入
}

// defaultLimiter 包级默认实例。用 atomic.Pointer 取代 mutex：只在 Init 写一次、
// 每次请求都读，指针的原子读写足够且去掉了读写锁互斥。
var defaultLimiter atomic.Pointer[Limiter]

// Init 构造包级默认实例。rdb 为限流计数所用的 Redis 客户端，须在 redis.Init 之后传入 redis.Client()。
func Init(rdb *goredis.Client) {
	// 移除了原先对包级 redis.Client() 的兜底，nil 必须在此拦下，否则要到请求时才 NPE，难定位。
	if rdb == nil {
		panic("ratelimiter: Init 收到 nil redis 客户端，请先 redis.Init 再传入 redis.Client()")
	}
	l := &Limiter{instanceID: newInstanceID(), rdb: rdb}
	defaultLimiter.Store(l)
	log.Printf("ratelimiter 已就绪: instanceID=%s", l.instanceID)
}

// RateLimiter 限流注解，按 limitType 决定全局/按 IP/按实例。零值取默认。
//
// 用法：ratelimiter.RateLimiter(60*time.Second, 10, ratelimiter.LimitTypeIP, 0, "")
func RateLimiter(window time.Duration, count int, limitType LimitType, timeout time.Duration, message string) gin.HandlerFunc {
	return handler(newOptions(window, count, limitType, timeout, message))
}

// RateLimiterWithKeyFunc 带动态维度的限流注解：fn 从请求里取维度值（如手机号），返回空串表示不带动态维度。零值取默认。
// 固定 LimitTypeGlobal：维度以 fn 的返回值为准，不再叠加 IP / 实例。
//
// 用法：ratelimiter.RateLimiterWithKeyFunc(60*time.Second, 1, func(c *gin.Context) string { return c.Query("phoneNumber") }, 0, "")
func RateLimiterWithKeyFunc(window time.Duration, count int, fn func(*gin.Context) string, timeout time.Duration, message string) gin.HandlerFunc {
	o := newOptions(window, count, LimitTypeGlobal, timeout, message)
	o.keyFunc = fn
	return handler(o)
}
