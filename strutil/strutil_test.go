package strutil

import (
	"reflect"
	"testing"
)

// TestParseIDList 逗号分隔的主键串切分。
func TestParseIDList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []int64
	}{
		{"单个", "1762000000000000001", []int64{1762000000000000001}},
		{"多个", "1,2,3", []int64{1, 2, 3}},
		{"含空白", " 1 , 2 ", []int64{1, 2}},
		// 超出 JS 安全整数范围的 ID 前端以字符串下发，ParseInt 不丢精度。
		{"雪花 ID 不丢精度", "1762000000000000001,1762000000000000002",
			[]int64{1762000000000000001, 1762000000000000002}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIDList(tt.in)
			if err != nil {
				t.Fatalf("ParseIDList(%q): %v", tt.in, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseIDList(%q) = %v, 期望 %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestParseIDListRejects 任一段非法即整体拒绝——静默丢弃会删成部分成功。
func TestParseIDListRejects(t *testing.T) {
	for _, in := range []string{
		"",                     // 空串
		",",                    // 只有分隔符
		"1,",                   // 尾随分隔符
		"1,abc",                // 混入非数字
		"1,0",                  // 0 非法主键
		"1,-2",                 // 负数
		"1,2;DROP x",           // 注入形态
		"99999999999999999999", // 溢出 int64
	} {
		t.Run(in, func(t *testing.T) {
			if got, err := ParseIDList(in); err == nil {
				t.Errorf("ParseIDList(%q) = %v, 应返回错误", in, got)
			}
		})
	}
}

// TestCapitalize 首字母大写。
func TestCapitalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"abc", "Abc"},
		{"Abc", "Abc"},
		{"中文", "中文"},
		{"aBC", "ABC"},
	}
	for _, tt := range tests {
		if got := Capitalize(tt.in); got != tt.want {
			t.Errorf("Capitalize(%q) = %q, 期望 %q", tt.in, got, tt.want)
		}
	}
}

// TestIsHTTP http(s) 链接判定。
func TestIsHTTP(t *testing.T) {
	yes := []string{"http://x", "https://x", "HTTP://X", "Https://x"}
	for _, s := range yes {
		if !IsHTTP(s) {
			t.Errorf("IsHTTP(%q) = false, 期望 true", s)
		}
	}
	no := []string{"", "ftp://x", "//x", "httpx"}
	for _, s := range no {
		if IsHTTP(s) {
			t.Errorf("IsHTTP(%q) = true, 期望 false", s)
		}
	}
}
