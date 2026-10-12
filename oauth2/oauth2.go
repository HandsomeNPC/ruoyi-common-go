// Package oauth2 三方账号授权登录。
//
// Go 生态无 JustAuth 这样的对应物：golang.org/x/oauth2 只覆盖 OAuth2 协议本身
// (拼授权地址、code 换令牌)，而「拿令牌去哪换用户资料、换回什么形状」不在
// RFC 6749 范围内——oauth2.Endpoint 只有 AuthURL/TokenURL，没有 userinfo。
// 故本包的分工是：
//
//	x/oauth2   AuthCodeURL / Exchange / client 认证方式协商
//	provider   端点 URL + userinfo 请求 + 字段映射，一个平台一个 provider_*.go
//	StateStore state 存取，由调用方注入(内存或 Redis)
//
// 加一个平台 = 新建 provider_<平台>.go 实现 provider 的两个方法、登记进
// builtinProviders，约 60 行。
//
// 用法：
//
//	m, err := oauth2.New(cfg, oauth2.NewRedisStore(rdb, "global:social_auth_codes:"))
//	url, err := m.AuthorizeURL(ctx, "gitee")
//	user, err := m.Callback(ctx, "gitee", code, state)
package oauth2

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	xoauth2 "golang.org/x/oauth2"
)

// 已接入的平台标识。取值即前端传来的 source，也是落库的 source 值。
const (
	SourceGitee  = "gitee"
	SourceGithub = "github"
	SourceMaxkey = "maxkey"
	SourceTopiam = "topiam"
)

// ErrUnsupportedSource 平台未接入或配置不全。
// 由调用方转成「xx平台账号暂不支持」。
var ErrUnsupportedSource = errors.New("oauth2: 不支持的第三方登录类型")

const (
	// httpTimeout 单次请求超时。不设的话三方网关抽风会把请求协程挂死。
	httpTimeout = 10 * time.Second
	// stateTTL 授权超时时间。
	stateTTL = 3 * time.Minute
)

// AuthUser 三方用户信息。
//
// 字段按 sys_social 的列取齐，不复用第三方库的用户结构：goth.User 没有
// Scope/TokenType，且用 ExpiresAt time.Time 表达有效期，而 expire_in 是秒数。
type AuthUser struct {
	// UUID 平台内的用户唯一标识。落 open_id，并与 Source 拼成 auth_id。
	UUID     string
	Source   string
	Username string
	Nickname string
	Avatar   string
	Email    string
	Token    AuthToken
}

// AuthToken 令牌信息，字段与 sys_social 的令牌列一一对应。
// 多数平台只填得起前几项，其余留零。
type AuthToken struct {
	AccessToken  string
	ExpireIn     int
	RefreshToken string
	Scope        string
	TokenType    string
	IDToken      string
	OpenID       string
	UnionID      string
	AccessCode   string
	Code         string
}

// Manager 一组已就绪平台的授权入口，并发安全。
type Manager struct {
	store StateStore
	// entries 已就绪平台。New 建好后只读，故免锁；仅含配置齐全者，
	// 所以「不在表里」等价于「不支持」。
	entries map[string]entry
}

// entry provider 与其配置成对缓存，让请求期的 lookup 不必再回读配置。
type entry struct {
	provider provider
	cfg      ProviderConfig
}

// New 按配置挑出可用平台。
//
// 配置不全即跳过：缺 clientId/clientSecret(私有化部署再加 serverUrl)
// 都归进「不支持该平台」，而非启动失败——三方登录本就是可选功能。
// 想在启动期拦住「填了一半」的配置，调 Config.Validate。
func New(cfg Config, store StateStore) (*Manager, error) {
	if store == nil {
		return nil, errors.New("oauth2: StateStore 不能为空")
	}

	entries := make(map[string]entry, len(builtinProviders))
	for source, p := range builtinProviders {
		pc, ok := cfg.Type[source]
		if !ok || !pc.configured() {
			continue
		}
		// 让 provider 自查其余必填项(如私有化部署的 serverUrl)，
		// 不合格的当场剔除，免得留到运行期才报错。
		if _, err := p.oauthConfig(pc); err != nil {
			log.Printf("[oauth2] 平台 %s 配置无效，已跳过: %v", source, err)
			continue
		}
		entries[source] = entry{provider: p, cfg: pc}
	}

	log.Printf("[oauth2] 已就绪: 可用平台=%v", slices.Sorted(maps.Keys(entries)))
	return &Manager{store: store, entries: entries}, nil
}

// lookup 取平台实现与其配置，未就绪时返回 ErrUnsupportedSource。
func (m *Manager) lookup(source string) (entry, error) {
	e, ok := m.entries[source]
	if !ok {
		return entry{}, ErrUnsupportedSource
	}
	return e, nil
}

// AuthorizeURL 生成授权跳转地址，并把 state 存起来待回调校验。
//
// state 是裸 UUID，前端原样回传，不做 base64/JSON 编解码。
func (m *Manager) AuthorizeURL(ctx context.Context, source string) (string, error) {
	e, err := m.lookup(source)
	if err != nil {
		return "", err
	}
	oc, err := e.provider.oauthConfig(e.cfg)
	if err != nil {
		return "", err
	}

	state := uuid.NewString()
	if err := m.store.Save(ctx, state, stateTTL); err != nil {
		return "", fmt.Errorf("oauth2: 缓存 state 失败: %w", err)
	}
	return oc.AuthCodeURL(state), nil
}

// Callback 执行三方登录回调：校验 state、用 code 换令牌、再换用户资料。
func (m *Manager) Callback(ctx context.Context, source, code, state string) (*AuthUser, error) {
	e, err := m.lookup(source)
	if err != nil {
		return nil, err
	}
	if code == "" {
		return nil, errors.New("oauth2: 三方登录 code 为空")
	}
	if state == "" {
		return nil, ErrIllegalState
	}
	if err := m.store.Consume(ctx, state); err != nil {
		return nil, err
	}

	oc, err := e.provider.oauthConfig(e.cfg)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: httpTimeout}
	// 把 client 交给 x/oauth2 走令牌交换，超时设置才对 Exchange 生效。
	tok, err := oc.Exchange(context.WithValue(ctx, xoauth2.HTTPClient, client), code)
	if err != nil {
		return nil, fmt.Errorf("oauth2: %s 换取令牌失败: %w", source, err)
	}

	user, err := e.provider.fetchUser(ctx, e.cfg, client, tok)
	if err != nil {
		return nil, err
	}
	if user.UUID == "" {
		// 拿不到平台内唯一标识就没法拼 auth_id，绑定关系无从建立。
		return nil, fmt.Errorf("oauth2: %s 未返回用户唯一标识", source)
	}
	return user, nil
}

// tokenFrom 把 xoauth2.Token 摊成 AuthToken。
//
// scope / id_token 不是 xoauth2.Token 的固定字段，只能从 Extra 里取
// (x/oauth2 把标准字段之外的键都塞进 raw)。
func tokenFrom(tok *xoauth2.Token) AuthToken {
	at := AuthToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
	}
	if s, ok := tok.Extra("scope").(string); ok {
		at.Scope = s
	}
	if s, ok := tok.Extra("id_token").(string); ok {
		at.IDToken = s
	}
	// Expiry 零值表示平台未给有效期(如 github)，此时 ExpireIn 留 0。
	if !tok.Expiry.IsZero() {
		if secs := int(time.Until(tok.Expiry).Seconds()); secs > 0 {
			at.ExpireIn = secs
		}
	}
	return at
}
