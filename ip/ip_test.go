package ip

import (
	"net/http/httptest"
	"testing"
)

// TestResolveClientIPHeaderPrecedence 代理头的优先级顺序。
func TestResolveClientIPHeaderPrecedence(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Real-IP", "2.2.2.2")
	r.Header.Set("Proxy-Client-IP", "3.3.3.3")
	r.Header.Set("X-Forwarded-For", "1.1.1.1")

	if got, want := ResolveClientIP(r), "1.1.1.1"; got != want {
		t.Errorf("X-Forwarded-For 优先级最高: got %q, want %q", got, want)
	}

	// 去掉最高优先级的头，应退到下一个。
	r.Header.Del("X-Forwarded-For")
	if got, want := ResolveClientIP(r), "2.2.2.2"; got != want {
		t.Errorf("应退到 X-Real-IP: got %q, want %q", got, want)
	}
	r.Header.Del("X-Real-IP")
	if got, want := ResolveClientIP(r), "3.3.3.3"; got != want {
		t.Errorf("应退到 Proxy-Client-IP: got %q, want %q", got, want)
	}
}

// TestResolveClientIPTakesFirstOfForwardedChain X-Forwarded-For 取链路第一段才是最初的客户端。
func TestResolveClientIPTakesFirstOfForwardedChain(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2, 3.3.3.3")

	if got, want := ResolveClientIP(r), "1.1.1.1"; got != want {
		t.Errorf("应取链路第一段: got %q, want %q", got, want)
	}
}

// TestResolveClientIPSkipsUnknown 部分代理拿不到真实 IP 时会写 "unknown" 字面量，必须跳过它继续找下一个来源。
func TestResolveClientIPSkipsUnknown(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "unknown")
	r.Header.Set("X-Real-IP", "2.2.2.2")

	if got, want := ResolveClientIP(r), "2.2.2.2"; got != want {
		t.Errorf("应跳过 unknown: got %q, want %q", got, want)
	}

	// 大小写不敏感，且链路里夹着 unknown 时取后面第一个有效值。
	r.Header.Set("X-Forwarded-For", "UNKNOWN, 4.4.4.4")
	if got, want := ResolveClientIP(r), "4.4.4.4"; got != want {
		t.Errorf("应跳过 UNKNOWN 取下一段: got %q, want %q", got, want)
	}
}

// TestResolveClientIPFallsBackToRemoteAddr 所有代理头都没有时回落到 RemoteAddr 并剥掉端口。
func TestResolveClientIPFallsBackToRemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "5.5.5.5:54321"
	if got, want := ResolveClientIP(r), "5.5.5.5"; got != want {
		t.Errorf("应回落到 RemoteAddr 并剥掉端口: got %q, want %q", got, want)
	}

	// IPv6 形如 "[::1]:8080"，剥端口后还要剥方括号。
	r.RemoteAddr = "[2001:db8::1]:8080"
	if got, want := ResolveClientIP(r), "2001:db8::1"; got != want {
		t.Errorf("IPv6 应剥掉端口与方括号: got %q, want %q", got, want)
	}

	// 没有端口段时原样使用。
	r.RemoteAddr = "6.6.6.6"
	if got, want := ResolveClientIP(r), "6.6.6.6"; got != want {
		t.Errorf("无端口时应原样返回: got %q, want %q", got, want)
	}
}

// TestResolveClientIPStripsIPv6Brackets 头里的 IPv6 字面量带方括号时要剥掉。
func TestResolveClientIPStripsIPv6Brackets(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "[2001:db8::1]")

	if got, want := ResolveClientIP(r), "2001:db8::1"; got != want {
		t.Errorf("应剥掉方括号: got %q, want %q", got, want)
	}
}

// TestResolveClientIPNilRequest 不 panic。
func TestResolveClientIPNilRequest(t *testing.T) {
	if got := ResolveClientIP(nil); got != "" {
		t.Errorf("nil 请求应返回空串, got %q", got)
	}
}
