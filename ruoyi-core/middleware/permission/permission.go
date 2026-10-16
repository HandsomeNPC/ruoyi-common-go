// Package permission sa-token 权限/角色校验中间件。
package permission

import (
	"log"

	sagin "github.com/HandsomeNPC/sa-token-go/integrations/gin"
	"github.com/gin-gonic/gin"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/errs"
	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/middleware/checklogin"
	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/model/response"
)

// 鉴权失败的提示文案。未登录文案由 checklogin 统一给出。
const msgNotPermission = "没有访问权限，请联系管理员授权"

// CheckPermission 权限码校验中间件，多个权限码任一命中即放行。
//
// 校验本身复用 sa-token（权限于登录时写入 account session），本层只做响应适配：
// sagin.CheckPermission 直接吐 gin.H 且 HTTP 状态非 200，与本项目 R + 恒 200 的契约不符，
// 故改用返回 error 的 ByToken 版本，交由 middleware.Recover 统一渲染。
//
// 语义为 OR（同 sagin.CheckPermission）。Java @SaCheckPermission 默认 AND，
// 但上游所有 Controller 都只传单个权限码，差异不可观测。
func CheckPermission(perms ...string) gin.HandlerFunc {
	return check(perms, sagin.CheckPermissionOrByToken, "权限码校验失败")
}

// CheckRole 角色校验中间件，多个角色任一命中即放行。
func CheckRole(roles ...string) gin.HandlerFunc {
	return check(roles, sagin.CheckRoleOrByToken, "角色权限校验失败")
}

// check 组装校验中间件。verify 失败即视为无权限，其内部区分不了的未登录场景由登录态预检兜住。
func check(want []string, verify func(string, []string) error, reason string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 权限校验前先判登录态：Java 的 @SaCheckPermission 内部先过登录校验，
		// 未登录抛 NotLoginException(401) 而非 NotPermissionException(403)。
		// sagin 的 ByToken 版只回"无权限"，不先判登录，故这里显式补上，
		// 否则 token 过期会错报 403，前端也就不会跳登录页。
		if !checklogin.Guard(c) {
			return
		}
		token := sagin.GetTokenFromCtx(c)
		if err := verify(token, want); err != nil {
			log.Printf("[satoken] 请求地址'%s',%s'%v': %v", c.Request.URL.Path, reason, want, err)
			_ = c.Error(errs.New(response.CodeForbidden, msgNotPermission, err.Error()))
			c.Abort()
			return
		}
		c.Next()
	}
}
