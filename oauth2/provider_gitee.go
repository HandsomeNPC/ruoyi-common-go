package oauth2

import (
	"context"
	"encoding/json"
	"net/http"

	xoauth2 "golang.org/x/oauth2"
)

// giteeProvider Gitee。端点写死：x/oauth2/endpoints 的平台表里没有 Gitee。
type giteeProvider struct{}

func (giteeProvider) oauthConfig(cfg ProviderConfig) (*xoauth2.Config, error) {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user_info"}
	}
	return &xoauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Scopes:       scopes,
		Endpoint: xoauth2.Endpoint{
			AuthURL:  "https://gitee.com/oauth/authorize",
			TokenURL: "https://gitee.com/oauth/token",
			// Gitee 要求 client 凭据放在参数体里，写死省掉 x/oauth2 的试探往返。
			AuthStyle: xoauth2.AuthStyleInParams,
		},
	}, nil
}

func (giteeProvider) fetchUser(ctx context.Context, _ ProviderConfig,
	client *http.Client, tok *xoauth2.Token) (*AuthUser, error) {

	// id 用 json.Number 收：Gitee 返回的是数字，而 auth_id/open_id 是字符串列。
	var u struct {
		ID     json.Number `json:"id"`
		Login  string      `json:"login"`
		Name   string      `json:"name"`
		Avatar string      `json:"avatar_url"`
		Email  string      `json:"email"`
	}
	// Gitee 认 access_token 查询参数，故不必额外设授权头。
	url := "https://gitee.com/api/v5/user?access_token=" + tok.AccessToken
	if err := getJSON(ctx, client, url, nil, &u); err != nil {
		return nil, err
	}

	return &AuthUser{
		UUID:     u.ID.String(),
		Source:   SourceGitee,
		Username: u.Login,
		Nickname: u.Name,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Token:    tokenFrom(tok),
	}, nil
}
