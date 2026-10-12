package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	xoauth2 "golang.org/x/oauth2"
)

// provider 一个三方平台的接入点。不导出：加平台在本包内进行，
// 外部要扩展应提 PR 而非注入——否则 Manager 还得为外部实现兜并发与配置校验。
type provider interface {
	// oauthConfig 构造交给 x/oauth2 的配置：端点地址与默认 scope 由各平台定。
	// serverUrl 非法等配置问题在此返回错误。
	oauthConfig(cfg ProviderConfig) (*xoauth2.Config, error)

	// fetchUser 用令牌换用户资料。RFC 6749 不覆盖这一步，故每个平台各自实现。
	fetchUser(ctx context.Context, cfg ProviderConfig,
		client *http.Client, tok *xoauth2.Token) (*AuthUser, error)
}

// builtinProviders 全部平台实现。与配置无关，是否可用由 New 按配置筛。
//
// 加平台：照 provider_gitee.go / provider_github.go(固定端点) 或
// provider_maxkey.go / provider_topiam.go(私有化部署，端点由 serverUrl 拼出)
// 新建一个 provider_<平台>.go，实现 provider 的两个方法，再在此登记。
var builtinProviders = map[string]provider{
	SourceGitee:  giteeProvider{},
	SourceGithub: githubProvider{},
	SourceMaxkey: maxkeyProvider{},
	SourceTopiam: topiamProvider{},
}

// getJSON 带令牌请求一个返回 JSON 的接口并解码进 dest。
// header 用于各平台不同的授权头形态(github 是 token xxx，OIDC 系是 Bearer xxx)。
func getJSON(ctx context.Context, client *http.Client, url string,
	header http.Header, dest any) error {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("oauth2: 构造请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("oauth2: 请求 %s 失败: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("oauth2: 读取响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// 带上响应体便于排查(令牌过期、scope 不足等平台都写在 body 里)。
		return fmt.Errorf("oauth2: %s 返回 %d: %s", url, resp.StatusCode, truncate(string(body), 200))
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("oauth2: 解析响应失败: %w, 响应: %s", err, truncate(string(body), 200))
	}
	return nil
}

// bearerHeader 构造 Authorization: Bearer <token> 头，OIDC 系平台通用。
func bearerHeader(accessToken string) http.Header {
	return http.Header{"Authorization": []string{"Bearer " + accessToken}}
}

// normalizeServerURL 校验私有化部署平台的 serverUrl 并剥掉尾部斜杠。
//
// 必须剥：端点由 base + "/oauth2/auth" 拼出，带尾斜杠会拼出 "//oauth2/auth"。
func normalizeServerURL(serverURL string) (string, error) {
	if serverURL == "" {
		return "", fmt.Errorf("serverUrl 未配置")
	}
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("serverUrl 须以 http:// 或 https:// 开头: %s", serverURL)
	}
	if u.Host == "" {
		return "", fmt.Errorf("serverUrl 缺少主机名: %s", serverURL)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("serverUrl 不应带查询串或片段: %s", serverURL)
	}
	return strings.TrimRight(serverURL, "/"), nil
}

// firstNonEmpty 返回首个非空串。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// truncate 截断过长的字符串，避免把整页 HTML 塞进日志。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
