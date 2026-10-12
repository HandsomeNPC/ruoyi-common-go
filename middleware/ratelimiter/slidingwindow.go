package ratelimiter

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// slidingWindowScript 滑动窗口限流，单次 EVAL 原子完成，避免"读-判-写"竞态。
//
// KEYS[1] 限流键(ZSET)
// ARGV[1] 当前时间(毫秒)  ARGV[2] 窗口(毫秒)
// ARGV[3] 窗口内允许次数  ARGV[4] 键存活时间(毫秒)  ARGV[5] 本次请求唯一成员
// 返回剩余次数；-1 表示超限。
var slidingWindowScript = goredis.NewScript(`
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit  = tonumber(ARGV[3])
local ttl    = tonumber(ARGV[4])
local member = ARGV[5]

-- 清掉窗口外的记录：权限在 window 毫秒后自动归还。
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

local used = redis.call('ZCARD', key)
if used >= limit then
  -- 超限也要续期，避免键在持续打压下提前过期导致计数被重置。
  redis.call('PEXPIRE', key, ttl)
  return -1
end

redis.call('ZADD', key, now, member)
redis.call('PEXPIRE', key, ttl)
return limit - used - 1
`)

// acquire 扣减一次额度，返回剩余次数；-1 表示超限。
func (l *Limiter) acquire(ctx context.Context, key string, o *options) (int64, error) {
	now := time.Now()
	// TTL 至少覆盖一个窗口，否则键会在窗口结束前消失导致限流失效。
	ttl := max(o.timeout, o.window)

	// 成员必须唯一，否则同一毫秒内的并发请求会因 ZADD 覆盖而只计一次。
	member := fmt.Sprintf("%d-%s", now.UnixNano(), newMemberSuffix())

	remain, err := slidingWindowScript.Run(ctx, l.rdb, []string{key},
		now.UnixMilli(), o.window.Milliseconds(), o.count, ttl.Milliseconds(), member,
	).Int64()
	if err != nil {
		return 0, fmt.Errorf("ratelimiter: 执行限流脚本失败: %w", err)
	}
	return remain, nil
}
