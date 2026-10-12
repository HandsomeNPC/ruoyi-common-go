package oauth2

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ErrIllegalState state 缺失、已过期或已被用过。
var ErrIllegalState = errors.New("oauth2: 三方登录 state 非法或已失效")

// StateStore state 的存取。state 的全部意义是防 CSRF，故必须单次有效。
//
// 本包自带内存与 Redis 两种实现，调用方也可自行实现(如换 etcd)。
type StateStore interface {
	// Save 存入 state，过期时间为 ttl。
	Save(ctx context.Context, state string, ttl time.Duration) error
	// Consume 取出并立即作废 state。
	// state 不存在(已过期/已用过/伪造)时返回 ErrIllegalState。
	Consume(ctx context.Context, state string) error
}

// memoryStore 进程内 state 存储。
//
// 仅适用于单进程部署：多进程下授权跳转与回调可能落在不同进程，
// 回调那边取不到 state 会一律报 ErrIllegalState。多进程用 RedisStore。
type memoryStore struct {
	mu     sync.Mutex
	states map[string]time.Time // state -> 过期时刻
}

// NewMemoryStore 建进程内 state 存储，适用于单进程部署。
func NewMemoryStore() StateStore {
	return &memoryStore{states: map[string]time.Time{}}
}

func (s *memoryStore) Save(_ context.Context, state string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 顺手清掉过期项。没有后台 goroutine 清理，否则包级状态会跟着进程常驻；
	// state 量级小(每次点三方登录才加一个)且 TTL 只有几分钟，写时清足够。
	now := time.Now()
	for k, exp := range s.states {
		if now.After(exp) {
			delete(s.states, k)
		}
	}
	s.states[state] = now.Add(ttl)
	return nil
}

func (s *memoryStore) Consume(_ context.Context, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.states[state]
	// 取出即删：即便已过期也要删，省得留着占内存。
	delete(s.states, state)
	if !ok || time.Now().After(exp) {
		return ErrIllegalState
	}
	return nil
}

// redisStore Redis state 存储，多进程部署可用。
type redisStore struct {
	rdb    *goredis.Client
	prefix string
}

// NewRedisStore 建 Redis state 存储。
// prefix 是键前缀(如 "global:social_auth_codes:")，由调用方给以免本包绑死键格式。
func NewRedisStore(rdb *goredis.Client, prefix string) StateStore {
	return &redisStore{rdb: rdb, prefix: prefix}
}

func (s *redisStore) Save(ctx context.Context, state string, ttl time.Duration) error {
	return s.rdb.Set(ctx, s.prefix+state, state, ttl).Err()
}

// Consume 用 GETDEL 原子地取值兼作废，并发重放只有一个能过。
func (s *redisStore) Consume(ctx context.Context, state string) error {
	err := s.rdb.GetDel(ctx, s.prefix+state).Err()
	if errors.Is(err, goredis.Nil) {
		return ErrIllegalState
	}
	if err != nil {
		return fmt.Errorf("oauth2: 校验 state 失败: %w", err)
	}
	return nil
}
