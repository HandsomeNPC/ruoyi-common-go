// Package audit 把登录用户身份写进 request context，供 repository 审计回调填充 create_by 等列。
package audit

import (
	"github.com/gin-gonic/gin"

	pkgrepo "github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/repository"
	lh "github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/utils/loginhelper"
)

// AuditContext 把登录用户的 userId/deptId 写进 request context，供 pkg/repository
// 的审计回调填 create_by/create_dept 等列。
//
// 之所以要这道中间件：GORM 回调只拿得到 *gorm.DB，取不到 *gin.Context，而 pkg/repository
// 也不能反向 import 本包（会成环），故登录态只能经 ctx 接力。Go 无全局 thread-local 等价物，
// 只能由这道中间件显式挂载。
//
// 须注册在 TokenInterceptor 之后：GetLoginUser 读的 token 由它写进 gin.Context。
// 未登录时不写，由回调按 -1 兜底。
//
// GetLoginUser 在 utils/loginhelper，与本包不同名，这里仍起别名 lh 以便与 pkg/repository 区分。
func AuditContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		if lu := lh.GetLoginUser(c); lu != nil {
			c.Request = c.Request.WithContext(
				pkgrepo.WithAuditUser(c.Request.Context(), pkgrepo.AuditUser{
					UserID: lu.UserID,
					DeptID: lu.DeptID,
				}))
		}
		c.Next()
	}
}
