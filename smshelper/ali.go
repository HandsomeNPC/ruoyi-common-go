package smshelper

import (
	"fmt"

	"github.com/HandsomeNPC/ruoyi-common-go/jsonx"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
)

// 阿里云接入点默认值：yaml 未显式配置时由 initAliClient 兜底。
const (
	defaultEndpoint = "dysmsapi.aliyuncs.com"
	defaultRegionID = "cn-hangzhou"
)

// aliyunSuccessCode 业务成功标志，响应 Body.Code 非 OK 即失败。
const aliyunSuccessCode = "OK"

// aliClient 包级默认客户端。类型是阿里云 SDK 对象，故放此文件，smshelper.go 不感知 dysmsapi。
// 未就绪时为 nil，SendByAli 返回 ErrDisabled。
var aliClient *dysmsapi.Client

// initAliClient 按阿里云凭证 + 接入点构造 dysmsapi 客户端。
//
// AK 缺失直接返回 error——空 AK 发出去必被厂商拒，不如启动期就标记该渠道不可用，
// 失败时 aliClient 保持 nil，Init 据此不计入 enabled。签名 / 模板是每次发信入参，不在此校验。
func initAliClient(a AliyunConfig) error {
	if a.Endpoint == "" {
		a.Endpoint = defaultEndpoint
	}
	if a.RegionID == "" {
		a.RegionID = defaultRegionID
	}
	if a.AccessKeyID == "" {
		return fmt.Errorf("smshelper: sms.aliyun.accessKeyId 未配置")
	}
	if a.AccessKeySecret == "" {
		return fmt.Errorf("smshelper: sms.aliyun.accessKeySecret 未配置")
	}

	akID, akSecret, endpoint, regionID := a.AccessKeyID, a.AccessKeySecret, a.Endpoint, a.RegionID
	cl, err := dysmsapi.NewClient(&openapiutil.Config{
		AccessKeyId:     &akID,
		AccessKeySecret: &akSecret,
		Endpoint:        &endpoint,
		RegionId:        &regionID,
	})
	if err != nil {
		return fmt.Errorf("smshelper: 构造阿里云客户端失败: %w", err)
	}
	aliClient = cl
	return nil
}

// SendByAli 用阿里云渠道发一条短信，未就绪时返回 ErrDisabled。
//
// templateParam 为模板变量，接受任意结构（如 map[string]string{"code": code}），
// 方法内序列化为 JSON 透传；键名须与阿里云模板里 ${xxx} 一致，由调用方保证。
// signName / templateCode 为空直接报错，不发到网关。
//
// dysmsapi SDK 的 SendSms 不收 ctx，且短信通常异步处理，故不做请求级取消——
// 真正的网关抽风由 SDK 默认读超时兜住。
func SendByAli(phone, signName, templateCode string, templateParam any) error {
	cl := aliClient
	if cl == nil {
		return ErrDisabled
	}
	if signName == "" || templateCode == "" {
		return fmt.Errorf("smshelper: signName / templateCode 未传入")
	}

	paramBytes, err := jsonx.Marshal(templateParam)
	if err != nil {
		return fmt.Errorf("smshelper: 序列化模板参数失败: %w", err)
	}
	templateParamStr := string(paramBytes)

	resp, err := cl.SendSms(&dysmsapi.SendSmsRequest{
		PhoneNumbers:  &phone,
		SignName:      &signName,
		TemplateCode:  &templateCode,
		TemplateParam: &templateParamStr,
	})
	if err != nil {
		return fmt.Errorf("smshelper: 请求短信网关失败: %w", err)
	}
	if resp == nil || resp.Body == nil || resp.Body.Code == nil {
		return fmt.Errorf("smshelper: 短信网关返回空响应")
	}
	if respCode := *resp.Body.Code; respCode != aliyunSuccessCode {
		// 厂商文案透出去：调用方要把它当作给前端的提示。文案为空时退回 Code。
		hint := respCode
		if resp.Body.Message != nil && *resp.Body.Message != "" {
			hint = *resp.Body.Message
		}
		return fmt.Errorf("smshelper: 短信发送失败: %s", hint)
	}
	return nil
}
