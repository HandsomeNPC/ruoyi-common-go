package constant

// 留在此处的符号都被 ruoyi-core 自己的包用到，故不能下移到业务模块：
// SuperAdminUserID / AllPermission 由 utils/loginhelper 用，
// ExcludeProperties 由 middleware/operlog 用。
// 其余 system 专属常量已迁至 internal/system/constant。
const (
	// SuperAdminUserID 超管用户ID，loginhelper 据此判定超管身份。
	SuperAdminUserID int64 = 1761100000000000001
	// AllPermission 超管的全部权限标识，前端 hasPermi 指令按字面量比对，不可改动。
	AllPermission = "*:*:*"
)

var ExcludeProperties = []string{"password", "oldPassword", "newPassword", "confirmPassword"}
