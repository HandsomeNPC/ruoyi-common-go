package oauth2

import (
	"context"
	"fmt"
	"net/http"

	xoauth2 "golang.org/x/oauth2"
)

// maxkeyProvider MaxKey 单点登录。
//
// 私有化部署：端点不是固定 URL，而是由配置的 serverUrl 拼出，故 serverUrl 必填。
type maxkeyProvider struct{}

func (maxkeyProvider) oauthConfig(cfg ProviderConfig) (*xoauth2.Config, error) {
	base, err := normalizeServerURL(cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("oauth2: 平台 maxkey 的 serverUrl 无效: %w", err)
	}
	return &xoauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		// 无默认 scope：MaxKey 的权限由服务端应用配置决定，不在授权地址上声明。
		Scopes: cfg.Scopes,
		Endpoint: xoauth2.Endpoint{
			AuthURL:  base + "/sign/authz/oauth/v20/authorize",
			TokenURL: base + "/sign/authz/oauth/v20/token",
			// client 凭据放参数体。这类差异由 x/oauth2 处理，不必手写签名。
			AuthStyle: xoauth2.AuthStyleInParams,
		},
	}, nil
}

func (maxkeyProvider) fetchUser(ctx context.Context, cfg ProviderConfig,
	client *http.Client, tok *xoauth2.Token) (*AuthUser, error) {

	base, err := normalizeServerURL(cfg.ServerURL)
	if err != nil {
		return nil, err
	}

	// 唯一标识三个候选键：MaxKey 各版本的 /me 返回键名不一致
	// (userId / userid / 回落到 OIDC 的 sub)，故收全再按优先级取。
	var u struct {
		UserID      string `json:"userId"`
		Userid      string `json:"userid"`
		Sub         string `json:"sub"`
		Username    string `json:"username"`
		DisplayName string `json:"displayName"`
		AvatarURL   string `json:"avatar_url"`
		Email       string `json:"email"`
	}
	endpoint := base + "/sign/api/oauth/v20/me"
	if err := getJSON(ctx, client, endpoint, bearerHeader(tok.AccessToken), &u); err != nil {
		return nil, err
	}

	return &AuthUser{
		UUID:     firstNonEmpty(u.UserID, u.Userid, u.Sub),
		Source:   SourceMaxkey,
		Username: u.Username,
		Nickname: u.DisplayName,
		Avatar:   u.AvatarURL,
		Email:    u.Email,
		Token:    tokenFrom(tok),
	}, nil
}
