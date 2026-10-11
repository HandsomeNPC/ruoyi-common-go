package smshelper

// SMSConfig 短信配置，对应 yaml 的 sms 段。
//
// 顶层只有总开关 enabled；aliyun 子段只放凭证 + 接入点（多业务共用一个客户端）。
// 签名 / 模板属于业务侧配置，按业务各自挂子段——captcha 子段即登录验证码模板，
// 由 captcha_handler 调 SendByAli 时透传。
type SMSConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Aliyun  AliyunConfig  `mapstructure:"aliyun"`
	Captcha CaptchaConfig `mapstructure:"captcha"`
}

// AliyunConfig 阿里云短信（国内，dysmsapi-2017-05-25）凭证与接入点。
//
// 只放建客户端要用的：AK / 接入点 / region。签名与模板是每次发信入参，
// 不在此处——多业务多模板时各业务自带。endpoint / regionId 留空由 Init 兜默认值。
type AliyunConfig struct {
	AccessKeyID     string `mapstructure:"accessKeyId"`
	AccessKeySecret string `mapstructure:"accessKeySecret"`
	Endpoint        string `mapstructure:"endpoint"`
	RegionID        string `mapstructure:"regionId"`
}

// CaptchaConfig 登录验证码短信的签名与模板。
//
// 新增别的短信业务（如通知类）照此再挂一个子段，各自调 SendByAli 透传即可。
type CaptchaConfig struct {
	SignName     string `mapstructure:"signName"`
	TemplateCode string `mapstructure:"templateCode"`
}
