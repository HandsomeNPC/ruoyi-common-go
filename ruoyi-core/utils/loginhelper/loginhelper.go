// Package loginhelper 登录鉴权助手
package loginhelper

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	sasession "github.com/HandsomeNPC/sa-token-go/core/session"
	sagin "github.com/HandsomeNPC/sa-token-go/integrations/gin"
	"github.com/HandsomeNPC/sa-token-go/stputil"
	"github.com/gin-gonic/gin"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/constant"
	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/enum"
	authmodel "github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/model"
)

// LoginUserKey token session 中 *LoginUser 的键。
const LoginUserKey = "loginUser"

// LoginOptions 单次登录的超时覆盖项，对应 sys_client 的 timeout / active_timeout 两列。
//
// 零值表示该项不覆盖、沿用 satoken 全局配置。需 satoken.dynamicActiveTimeout=true
// 才会生效，否则 sa-token-go 会丢弃 ActiveTimeout 并打告警。
type LoginOptions struct {
	// Device 设备类型，对应 SaLoginParameter.deviceType，"" 走默认。
	Device string
	// Timeout token 固定有效期(秒)，0 表示沿用全局配置。
	Timeout int64
	// ActiveTimeout token 最低活跃频率(秒)，0 表示沿用全局配置。
	ActiveTimeout int64
}

// Login 签发令牌、把 *LoginUser 存入 token session，并回填 loginUser.Token。
// device 对应 SaLoginParameter.deviceType，"" 走默认。
func Login(loginUser *authmodel.LoginUser, device string) (string, error) {
	return LoginWithOptions(loginUser, LoginOptions{Device: device})
}

// LoginWithOptions 按客户端超时签发令牌，其余同 Login。
//
// RuoYi 的超时是按 sys_client 配的，各客户端(pc / app / 小程序)可不同，
// 故须随每次登录传入而非只读全局配置。
func LoginWithOptions(loginUser *authmodel.LoginUser, opts LoginOptions) (string, error) {
	loginID, err := loginUser.LoginID()
	if err != nil {
		return "", err
	}

	param := sagin.NewLoginParameter()
	param.Device = opts.Device
	param.Timeout = opts.Timeout
	if opts.ActiveTimeout != 0 {
		param.SetActiveTimeout(opts.ActiveTimeout)
	}

	token, err := sagin.LoginWithParameter(loginID, param)
	if err != nil {
		return "", err
	}
	loginUser.Token = token

	sess, err := stputil.GetTokenSessionOrCreate(token)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(loginUser)
	if err != nil {
		return "", err
	}
	// 必须显式带 TTL：session.Set 省略 TTL 时会以「无过期」落库，
	// 令 token 过期后这份 LoginUser 仍长留 Redis。
	if err := setWithTokenTTL(sess, token, LoginUserKey, string(payload)); err != nil {
		return "", err
	}
	if err := stputil.SetPermissions(loginID, toSaTokenPerms(loginUser.MenuPermission)); err != nil {
		return "", err
	}
	if err := stputil.SetRoles(loginID, loginUser.RolePermission); err != nil {
		return "", err
	}
	return token, nil
}

// setWithTokenTTL 以 token 的剩余有效期为 TTL 写 session，使两者同步失效。
// token 被配成永不过期(TTL=-1)时不带 TTL，与 token 行为保持一致。
func setWithTokenTTL(sess *sasession.Session, token, key, value string) error {
	ttl, err := sagin.GetManager().GetTokenTimeout(token)
	if err != nil {
		return err
	}
	if ttl > 0 {
		return sess.Set(key, value, time.Duration(ttl)*time.Second)
	}
	return sess.Set(key, value)
}

// toSaTokenPerms 把超管的 *:*:* 换成 sa-token 认得的 *，其余原样。
func toSaTokenPerms(perms []string) []string {
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		if p == constant.AllPermission {
			p = "*"
		}
		out = append(out, p)
	}
	return out
}

// loginUserCacheKey gin.Context 中本请求登录用户快照的键。
const loginUserCacheKey = "ruoyi:loginUser"

// GetLoginUser 取当前请求的登录用户，未登录返回 nil。
func GetLoginUser(c *gin.Context) *authmodel.LoginUser {
	if v, ok := c.Get(loginUserCacheKey); ok {
		// 未登录时存的是 nil，断言失败正好落回 nil。
		lu, _ := v.(*authmodel.LoginUser)
		return lu
	}
	lu := GetLoginUserByToken(sagin.GetTokenFromCtx(c))
	c.Set(loginUserCacheKey, lu)
	return lu
}

// GetLoginUserByToken 按指定 token 取登录用户。
func GetLoginUserByToken(token string) *authmodel.LoginUser {
	if token == "" {
		return nil
	}
	sess, err := sagin.GetTokenSession(token)
	if err != nil || sess == nil {
		return nil
	}
	v := sess.GetString(LoginUserKey)
	if v == "" {
		return nil
	}
	var lu authmodel.LoginUser
	if err := json.Unmarshal([]byte(v), &lu); err != nil {
		return nil
	}
	return &lu
}

// GetLoginID 取当前 token 的 loginID（形如 "sys_user:123"），未登录返回 ""。
func GetLoginID(c *gin.Context) string {
	token := sagin.GetTokenFromCtx(c)
	if token == "" {
		return ""
	}
	id, err := sagin.GetLoginID(token)
	if err != nil {
		return ""
	}
	return id
}

// GetUserID 取当前登录用户 ID，未登录返回 0。
func GetUserID(c *gin.Context) int64 {
	if lu := GetLoginUser(c); lu != nil {
		return lu.UserID
	}
	return 0
}

// GetUserIDStr 取当前登录用户 ID 字符串，未登录返回 ""。
func GetUserIDStr(c *gin.Context) string {
	if id := GetUserID(c); id != 0 {
		return strconv.FormatInt(id, 10)
	}
	return ""
}

// GetUsername 取当前登录用户名，未登录返回 ""。
func GetUsername(c *gin.Context) string {
	if lu := GetLoginUser(c); lu != nil {
		return lu.Username
	}
	return ""
}

// GetDeptID 取当前登录用户部门 ID，未登录返回 0。
func GetDeptID(c *gin.Context) int64 {
	if lu := GetLoginUser(c); lu != nil {
		return lu.DeptID
	}
	return 0
}

// GetDeptName 取当前登录用户部门名，未登录返回 ""。
func GetDeptName(c *gin.Context) string {
	if lu := GetLoginUser(c); lu != nil {
		return lu.DeptName
	}
	return ""
}

// GetDeptCategory 取当前登录用户部门类别编码，未登录返回 ""。
func GetDeptCategory(c *gin.Context) string {
	if lu := GetLoginUser(c); lu != nil {
		return lu.DeptCategory
	}
	return ""
}

// GetUserType 取当前登录用户类型，未登录返回 ""。
func GetUserType(c *gin.Context) string {
	if lu := GetLoginUser(c); lu != nil {
		return lu.UserType
	}
	return ""
}

// IsSuperAdmin 判断指定用户 ID 是否超级管理员。
func IsSuperAdmin(userID int64) bool {
	return userID == constant.SuperAdminUserID
}

// IsCurrentSuperAdmin 判断当前登录用户是否超级管理员。
func IsCurrentSuperAdmin(c *gin.Context) bool {
	return IsSuperAdmin(GetUserID(c))
}

// IsLogin 判断当前请求是否携带有效登录态。
func IsLogin(c *gin.Context) bool {
	token := sagin.GetTokenFromCtx(c)
	return token != "" && sagin.IsLogin(token)
}

// LogoutUser 注销指定用户的会话：角色/授权变动后让受影响在线用户下次请求即失效。
func LogoutUser(userID int64) {
	if userID <= 0 {
		return
	}
	loginID := enum.UserTypeSys.Code + ":" + strconv.FormatInt(userID, 10)
	if err := sagin.Logout(loginID); err != nil {
		log.Printf("[satoken] 注销用户 %d 会话失败: %v", userID, err)
	}
}

// LogoutUsers 批量注销用户会话，逐个调用 LogoutUser。
// 不做去重：sys_user_role 不会给同一用户重复行，调用方传入的集合语义上也无重复预期。
func LogoutUsers(userIDs []int64) {
	for _, id := range userIDs {
		LogoutUser(id)
	}
}
