package enum

// UserType 用户类型。
type UserType struct {
	Code string
}

// 用户类型枚举实例。
var (
	UserTypeSys = UserType{Code: "sys_user"}
	UserTypeApp = UserType{Code: "app_user"}
)
