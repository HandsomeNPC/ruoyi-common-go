package useragent

import "testing"

// TestParse 空/非法 UA 返回空串，有效 UA 返回非空浏览器名与 OS。
func TestParse(t *testing.T) {
	if b, o := Parse(""); b != "" || o != "" {
		t.Errorf("空串期望空, got browser=%q os=%q", b, o)
	}

	const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	b, o := Parse(ua)
	if b == "" {
		t.Error("有效 UA 期望非空浏览器名")
	}
	if o == "" {
		t.Error("有效 UA 期望非空 OS")
	}
}
