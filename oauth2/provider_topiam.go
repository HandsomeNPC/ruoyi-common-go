package oauth2

import (
	"context"
	"fmt"
	"net/http"

	xoauth2 "golang.org/x/oauth2"
)

// topiamProvider TOPIAM 身份管理平台。
//
// 私有化部署：端点不是固定 URL，而是由配置的 serverUrl 拼出，故 serverUrl 必填。
// 走标准 OIDC，用户资料字段即 OIDC 标准 claims。
type topiamProvider struct{}

func (topiamProvider) oauthConfig(cfg ProviderConfig) (*xoauth2.Config, error) {
	base, err := normalizeServerURL(cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("oauth2: 平台 topiam 的 serverUrl 无效: %w", err)
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		// OIDC 要求 openid；profile/email 换 userinfo 的昵称与邮箱。
		scopes = []string{"openid", "profile", "email"}
	}
	return &xoauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Scopes:       scopes,
		Endpoint: xoauth2.Endpoint{
			AuthURL:  base + "/oauth2/auth",
			TokenURL: base + "/oauth2/token",
			// client 凭据走 HTTP Basic。这类差异由 x/oauth2 处理，不必手写签名。
			AuthStyle: xoauth2.AuthStyleInHeader,
		},
	}, nil
}

func (topiamProvider) fetchUser(ctx context.Context, cfg ProviderConfig,
	client *http.Client, tok *xoauth2.Token) (*AuthUser, error) {

	base, err := normalizeServerURL(cfg.ServerURL)
	if err != nil {
		return nil, err
	}

	var u struct {
		Sub               string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Nickname          string `json:"nickname"`
		Name              string `json:"name"`
		Picture           string `json:"picture"`
		Email             string `json:"email"`
	}
	endpoint := base + "/oauth2/userinfo"
	if err := getJSON(ctx, client, endpoint, bearerHeader(tok.AccessToken), &u); err != nil {
		return nil, err
	}

	return &AuthUser{
		// sub 是 OIDC 规定的唯一标识，必有。
		UUID:     u.Sub,
		Source:   SourceTopiam,
		Username: u.PreferredUsername,
		Nickname: firstNonEmpty(u.Nickname, u.Name, u.PreferredUsername),
		Avatar:   u.Picture,
		Email:    u.Email,
		Token:    tokenFrom(tok),
	}, nil
}
