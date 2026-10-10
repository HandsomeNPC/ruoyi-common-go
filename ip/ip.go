// Package ip 提供从 HTTP 请求识别客户端 IP 与离线 IP 归属地解析的工具。
package ip

import (
	"net"
	"net/http"
	"strings"
)

// clientIPHeaders 依次尝试的代理头，取到第一个非空且不为 unknown 的就停。
var clientIPHeaders = []string{
	"X-Forwarded-For",
	"X-Real-IP",
	"Proxy-Client-IP",
	"WL-Proxy-Client-IP",
	"HTTP_CLIENT_IP",
	"HTTP_X_FORWARDED_FOR",
}

// ResolveClientIP 解析请求的客户端真实 IP。
// r: HTTP 请求，nil 时返回空串。
// 按 clientIPHeaders 顺序读代理头，都取不到再回落 RemoteAddr 并剥端口与方括号。
// 返回值依赖 nginx 覆写代理头才可信，因这些头本身可伪造。
func ResolveClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	for _, h := range clientIPHeaders {
		if ip := firstValidIP(r.Header.Get(h)); ip != "" {
			return ip
		}
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return stripIPBrackets(host)
	}
	return stripIPBrackets(r.RemoteAddr)
}

// firstValidIP 从逗号分隔的头值取第一个有效 IP，空或 unknown 跳过；都没有则返回空串。
func firstValidIP(value string) string {
	if value == "" {
		return ""
	}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" || strings.EqualFold(part, "unknown") {
			continue
		}
		return stripIPBrackets(part)
	}
	return ""
}

// stripIPBrackets 去掉 IPv6 字面量两端的方括号，如 "[::1]" -> "::1"。
func stripIPBrackets(ip string) string {
	return strings.Trim(strings.TrimSpace(ip), "[]")
}
