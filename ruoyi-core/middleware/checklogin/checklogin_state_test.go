package checklogin

import (
	"context"
	"os"
	"testing"
	"time"

	sagin "github.com/HandsomeNPC/sa-token-go/integrations/gin"
	saredis "github.com/HandsomeNPC/sa-token-go/storage/redis"
	"github.com/redis/go-redis/v9"
)

// 本文件的用例要用真实 Redis 建会话，才能验证被踢/被顶号的失效原因映射。
// 连不上就跳过，不让缺 Redis 的环境误报失败。
//
// 地址/密码走环境变量：
//
//	REDIS_ADDR（默认 127.0.0.1:6379）、REDIS_PASSWORD
//
// 固定用 DB 15 作测试库，避免踩业务数据。
func setupMgr(t *testing.T) {
	t.Helper()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       15,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		t.Skipf("跳过：连不上 Redis %s (%v)", addr, err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	mgr := sagin.NewBuilder().
		Storage(saredis.NewStorageFromClient(rdb)).
		TokenName("Authorization").
		TokenStyle(sagin.TokenStyleJWT).
		JwtSecretKey("test-secret-key-for-checklogin-only").
		Timeout(600).
		ActiveTimeout(600).
		IsConcurrent(false).
		IsShare(false). // 关并发且不共享 token => 同账号二次登录触发顶号
		Build()
	sagin.SetManager(mgr)
}

func TestVerifyKickOut(t *testing.T) {
	setupMgr(t)
	tok, err := sagin.Login("sys_user:9001")
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if msg, ok := Verify(tok); !ok {
		t.Fatalf("新登录应有效, 却得 %q", msg)
	}
	if err := sagin.Kickout("sys_user:9001"); err != nil {
		t.Fatalf("踢人失败: %v", err)
	}
	msg, ok := Verify(tok)
	if ok {
		t.Fatal("被踢后应失效")
	}
	if msg != msgKickOut {
		t.Errorf("被踢文案应为 %q, 实际 %q", msgKickOut, msg)
	}
	t.Logf("KICK_OUT -> %q", msg)
}

func TestVerifyBeReplaced(t *testing.T) {
	setupMgr(t)
	tok, err := sagin.Login("sys_user:9002", "pc")
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	// 用显式 ReplacedByToken：非并发登录时 sa-token-go 内部走的是 kickout 而非
	// Replaced，同账号二次登录并不会产出 BE_REPLACED 标记。
	if err := sagin.GetManager().ReplacedByToken(tok); err != nil {
		t.Fatalf("顶号失败: %v", err)
	}
	msg, ok := Verify(tok)
	if ok {
		t.Fatal("被顶号后旧 token 应失效")
	}
	if msg != msgBeReplaced {
		t.Errorf("顶号文案应为 %q, 实际 %q", msgBeReplaced, msg)
	}
	t.Logf("BE_REPLACED -> %q", msg)
}

func TestVerifyGarbageToken(t *testing.T) {
	setupMgr(t)
	msg, ok := Verify("totally-not-a-valid-jwt")
	if ok {
		t.Fatal("伪造 token 应失效")
	}
	t.Logf("无效 token -> %q", msg)
}
