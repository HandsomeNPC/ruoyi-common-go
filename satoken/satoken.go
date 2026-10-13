// Package satoken 封装 sa-token-go 的初始化与全局访问。
package satoken

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/sa-tokens/sa-token-go/core/listener"
	sagin "github.com/sa-tokens/sa-token-go/integrations/gin"
	saredis "github.com/sa-tokens/sa-token-go/storage/redis"
)

// defaultTokenName 读取 token 的请求头名默认值。
const defaultTokenName = "Authorization"

// onlineTokenKeyPrefix 在线用户摘要的 Redis key 前缀。
// 会话存储自己的键空间。项目自管的在线摘要键是另一份
// (internal/auth/service/syslogin.OnlineTokenKeyPrefix)，两者同值但互不依赖。
const onlineTokenKeyPrefix = "online_tokens:"

// errMissingSecretKey 密钥未配置。随机兜底会让多进程各签各的、重启后旧 token 全失效。
var errMissingSecretKey = errors.New("satoken.jwtSecretKey 未配置")

// Init 按传入的 SATokenConfig 与 Redis 客户端构建全局 sa-token Manager。
// Redis 客户端由调用方注入（不直接 import redis 包，避免与各进程的 redis 初始化耦合）；
// 须在 redis.Init 之后调用；配置非法直接 panic，fail-fast。
func Init(cfg SATokenConfig, rdb *redis.Client) {
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		panic(fmt.Errorf("satoken: 初始化失败: %w", err))
	}

	storage := saredis.NewStorageFromClient(rdb)

	manager := sagin.NewBuilder().
		Storage(storage).
		TokenName(cfg.TokenName).        // 可配
		IsConcurrent(cfg.IsConcurrent).  // 可配
		IsShare(cfg.IsShare).            // 可配
		TokenStyle(sagin.TokenStyleJWT). // 固定 JWT
		JwtSecretKey(cfg.JwtSecretKey).  // 可配
		IsReadCookie(true).              // 固定：启用 Cookie 读 token
		IsReadBody(true).                // 固定：启用 Body 读 token
		Build()

	sagin.SetManager(manager)
	// 登出/被踢/被顶时清 online_tokens 摘要。
	// sa-token-go 在这些动作内部同步触发事件，故无需再于各调用点手删。
	registerOnlineTokenCleanup(manager, rdb)
	log.Printf("[satoken] 已就绪: tokenName=%s", cfg.TokenName)
}

// registerOnlineTokenCleanup 在会话终结事件上删 online_tokens:<token>。
func registerOnlineTokenCleanup(mgr *sagin.Manager, rdb *redis.Client) {
	del := func(data *listener.EventData) {
		if data.Token == "" {
			return
		}
		key := onlineTokenKeyPrefix + data.Token
		if err := rdb.Del(context.Background(), key).Err(); err != nil {
			log.Printf("[satoken] 清理在线摘要 %s 失败: %v", key, err)
		}
	}
	mgr.RegisterFunc(listener.EventLogout, del)
	mgr.RegisterFunc(listener.EventKickout, del)
	mgr.RegisterFunc(listener.EventReplaced, del)
}

// Manager 返回全局 Manager，未调用 Init 会由 sagin.GetManager 自身 panic。
func Manager() *sagin.Manager {
	return sagin.GetManager()
}

// TokenName 返回生效的 token 请求头名，须在 Init 之后调用。
//
// 调用方(repeatsubmit、SSE 的 query token 规范化)需要的是 Init 兜底后的值，
// 而默认值由本包内部补齐、不回写调用方持有的 SATokenConfig，故不能直接读
// cfg.TokenName——yaml 省略该项时那边仍是空串。
func TokenName() string {
	return Manager().GetConfig().TokenName
}
