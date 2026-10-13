package operlog

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestLimitStaysWithinBudget 截断后输出不得超过 maxLen 个字符（小列列宽兜底）。
func TestLimitStaysWithinBudget(t *testing.T) {
	cases := []struct {
		name   string
		s      string
		maxLen int
	}{
		{"ascii超长", strings.Repeat("a", 300), 128},
		{"中文超长", strings.Repeat("中", 300), 128},
		{"4000大列", strings.Repeat("中", 5000), 3800},
		{"client_key列宽", strings.Repeat("x", 60), 32},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := limit(tc.s, tc.maxLen)
			if rc := utf8.RuneCountInString(got); rc > tc.maxLen {
				t.Fatalf("输出 %d rune 超过 maxLen %d", rc, tc.maxLen)
			}
		})
	}
}

// TestLimitNoTruncationShortInput 短输入原样返回，不带标记。
func TestLimitNoTruncationShortInput(t *testing.T) {
	if got := limit("abc", 128); got != "abc" {
		t.Fatalf("短输入应原样返回: got %q", got)
	}
	if got := limit(strings.Repeat("中", 100), 128); got != strings.Repeat("中", 100) {
		t.Fatalf("rune 数未超应原样返回")
	}
}

// TestLimitMarksTruncation 截断时带标记。
func TestLimitMarksTruncation(t *testing.T) {
	got := limit(strings.Repeat("a", 300), 128)
	if !strings.HasSuffix(got, truncatedSuffix) {
		t.Fatalf("截断应带标记: got %q", got)
	}
}

// TestRemoveSensitiveFieldsCaseInsensitive 键名大小写不敏感脱敏。
func TestRemoveSensitiveFieldsCaseInsensitive(t *testing.T) {
	fields := []string{"password", "oldPassword"}
	node := map[string]any{
		"password":    "secret",
		"Password":    "secret",
		"PASSWORD":    "secret",
		"oldPassword": "old",
		"OLDPASSWORD": "old",
		"name":        "bob",
		"nested": map[string]any{
			"password": "nested-secret",
		},
	}
	removeSensitiveFields(node, fields)

	for _, key := range []string{"password", "Password", "PASSWORD", "oldPassword", "OLDPASSWORD"} {
		if hasKey(node, key) {
			t.Errorf("大小写变体 %q 未脱敏", key)
		}
	}
	if _, ok := node["name"].(string); !ok {
		t.Errorf("非敏感字段不应被删")
	}
	nested, _ := node["nested"].(map[string]any)
	if _, ok := nested["password"]; ok {
		t.Errorf("嵌套层未脱敏")
	}
}

// hasKey 判断 map 顶层是否含某 key。
func hasKey(m map[string]any, key string) bool {
	for k := range m {
		if k == key {
			return true
		}
	}
	return false
}
