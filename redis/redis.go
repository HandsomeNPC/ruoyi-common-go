// Package redis Redis 客户端初始化与包级默认实例访问。
package redis

import (
	"fmt"
	"log"
	"sync/atomic"

	goredis "github.com/redis/go-redis/v9"
)

// 写只在启动期 Init 一次（外加关停时 Close 清空一次），读是每条 Redis 调用都走的
// 热路径，故用 atomic.Pointer 而非 RWMutex：读是一次原子 load、无锁；原子读写
// 自带 happens-before，关停清空与在途读不会竞争。
var clientHolder atomic.Pointer[goredis.Client]

// Init 按配置建立客户端并设置为包级默认实例，须在配置加载之后、开始接请求之前调用。
// Redis 是硬依赖（会话/缓存/锁/验证码全靠它），构造或探活失败直接 panic，fail-fast。
func Init(c RedisConfig) {
	client, err := newRedisClient(c)
	if err != nil {
		panic(fmt.Errorf("redis: 初始化失败: %w", err))
	}
	// 原子换入:若已存在旧实例(误重复 Init),先关其连接池避免泄漏。
	if old := clientHolder.Swap(client); old != nil {
		_ = old.Close()
	}
	log.Printf("[redis] 已连接 %s:%d db=%d", c.Host, c.Port, c.DB)
}

// Client 返回包级默认实例，未调用 Init 会 panic。
func Client() *goredis.Client {
	client := clientHolder.Load()
	if client == nil {
		panic("redis: 尚未初始化，请先调用 redis.Init")
	}
	return client
}

// Close 关闭并清空包级默认实例，供进程退出时 defer 调用。
func Close() {
	client := clientHolder.Swap(nil)
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		log.Printf("关闭 Redis 连接失败: %v", err)
	}
}
