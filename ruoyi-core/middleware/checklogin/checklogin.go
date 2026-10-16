// Package checklogin 登录态校验中间件。
package checklogin

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"

	satokencore "github.com/HandsomeNPC/sa-token-go/core"
	sagin "github.com/HandsomeNPC/sa-token-go/integrations/gin"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/errs"
	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/model/response"
)

// 登录态失效的提示文案，与 Java SaTokenExceptionHandler 按 NotLoginException.getType()
// 分支的四种文案一一对应。
const (
	msgTimeout    = "登录已过期，请重新登录"          // TOKEN_TIMEOUT / TOKEN_FREEZE
	msgBeReplaced = "当前账号已在其他设备登录，您已被强制下线" // BE_REPLACED
	msgKickOut    = "账号已被管理员强制下线"          // KICK_OUT
	msgDefault    = "登录状态异常，请重新登录"         // 其余情形
)

// CheckLogin 登录态校验中间件，对应 Java 的 @SaCheckLogin。
//
// 不直接用 sagin.CheckLogin()：它自行 c.JSON 出 {code,message,error} 且 HTTP 状态非 200，
// 与本项目 R + 恒 200 的契约不符——前端 axios 只在 2xx 分支里处理 code==401 跳登录页，
// 真 401 会落到 error 分支，只弹一个 toast 而不跳转。
func CheckLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !Guard(c) {
			return
		}
		c.Next()
	}
}

// Guard 校验当前请求的登录态，失效时登记 401 业务错误并 Abort，返回 false。
// 供 permission 等需要在自身逻辑前先过登录校验的中间件复用。
func Guard(c *gin.Context) bool {
	// token 由组级 TokenInterceptor 解析后写入 ctx，未挂该中间件时恒为空。
	msg, ok := Verify(sagin.GetTokenFromCtx(c))
	if ok {
		return true
	}
	log.Printf("[satoken] 请求地址'%s',认证失败'%s',无法访问系统资源", c.Request.URL.Path, msg)
	_ = c.Error(errs.New(response.CodeUnauthorized, msg, "登录态校验失败"))
	c.Abort()
	return false
}

// Verify 校验 token 的登录态，失败时返回对应的提示文案。
//
// 用 CheckLoginWithState 而非 IsLogin/CheckLogin：后两者把被踢、被顶号、冻结统一压成
// ErrNotLogin，拿不到失效原因，无法复刻 Java 的分支文案。供 SSE 等自行写响应的场景复用。
func Verify(token string) (msg string, ok bool) {
	if token == "" {
		return msgDefault, false
	}

	// CheckLoginWithState 对空 token 返回 (false, nil)，已在上面拦掉，故此处 err 必非 nil。
	logged, err := sagin.GetManager().CheckLoginWithState(token)
	if logged {
		return "", true
	}

	switch {
	case errors.Is(err, satokencore.ErrKickedOut):
		return msgKickOut, false
	case errors.Is(err, satokencore.ErrTokenReplaced):
		return msgBeReplaced, false
	// 固定有效期到点时 Redis 键直接消失，sa-token-go 分不出"过期"与"从未存在"，
	// 两者都落到 ErrInvalidTokenData；活跃超时(冻结)则有独立哨兵。
	// 二者在 Java 侧同为"登录已过期"，合并处理不损失语义。
	case errors.Is(err, satokencore.ErrActiveTimeout),
		errors.Is(err, satokencore.ErrInvalidTokenData):
		return msgTimeout, false
	default:
		return msgDefault, false
	}
}
