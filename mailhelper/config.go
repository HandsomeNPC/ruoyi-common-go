package mailhelper

// MailConfig 对应 yaml 的 mail 段。只为发验证码而存在，故不含附件/HTML/多收件人；
// 默认值与校验在 Init 兜底，不挂在结构体方法上。
type MailConfig struct {
	// Enabled 是否开启邮件功能。关闭时发信接口返回提示而非报错。
	Enabled bool `mapstructure:"enabled"`
	// Host SMTP 服务器地址，如 smtp.qq.com。
	Host string `mapstructure:"host"`
	// Port SMTP 端口：465 配 ssl=true，587 配 ssl=false（STARTTLS）。
	// 未配置时 Init 兜 465（SSL 直连的约定端口）。
	Port int `mapstructure:"port"`
	// From 发件地址，同时用作信封发件人。
	From string `mapstructure:"from"`
	// User SMTP 登录账号，多数服务商即完整邮箱地址。
	User string `mapstructure:"user"`
	// Password SMTP 登录密码。QQ/163 等要填「授权码」而非登录密码。
	Password string `mapstructure:"password"`
	// SSL true 走 SSL 直连（465），false 走 STARTTLS 升级（587）。
	// yaml 须显式写明——bool 零值即 false，无法在 Init 区分「未设」与「显式 false」。
	SSL bool `mapstructure:"ssl"`
}
