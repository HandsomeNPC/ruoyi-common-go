package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// 超时默认值：yaml 未配置时由 newRedisClient 兜底。
const (
	defaultDialTimeout  = 5 * time.Second
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second

	pingTimeout = 5 * time.Second
)

// newRedisClient 按配置建立 Redis 客户端并完成连接池设置与探活。
// host/port/db 缺失或 minIdleConns 越界直接返回 error，启动期即暴露；
// PoolSize/MinIdleConns 为 0 时交 go-redis 自带默认值。
func newRedisClient(c RedisConfig) (*goredis.Client, error) {
	if c.Host == "" {
		return nil, fmt.Errorf("redis: redis.host 未配置")
	}
	if c.Port <= 0 {
		return nil, fmt.Errorf("redis: redis.port 必须大于 0")
	}
	if c.DB < 0 {
		return nil, fmt.Errorf("redis: redis.db 不能为负数")
	}
	if c.PoolSize > 0 && c.MinIdleConns > c.PoolSize {
		return nil, fmt.Errorf("redis: redis.minIdleConns 不能大于 poolSize")
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	client := goredis.NewClient(&goredis.Options{
		Addr:            addr,
		Password:        c.Password,
		DB:              c.DB,
		ClientName:      c.ClientName,
		PoolSize:        c.PoolSize,
		MinIdleConns:    c.MinIdleConns,
		DialTimeout:     msOrDefault(c.DialTimeoutMs, defaultDialTimeout),
		ReadTimeout:     msOrDefault(c.ReadTimeoutMs, defaultReadTimeout),
		WriteTimeout:    msOrDefault(c.WriteTimeoutMs, defaultWriteTimeout),
		ConnMaxIdleTime: time.Duration(c.ConnMaxIdleTime) * time.Second,
	})

	if err := ping(client); err != nil {
		// 探活失败也要回收连接池。
		_ = client.Close()
		return nil, fmt.Errorf("redis: 连接 %s db=%d 失败: %w", addr, c.DB, err)
	}
	return client, nil
}

func ping(client *goredis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("探活失败: %w", err)
	}
	return nil
}

func msOrDefault(ms int, fallback time.Duration) time.Duration {
	if ms <= 0 {
		return fallback
	}
	return time.Duration(ms) * time.Millisecond
}
