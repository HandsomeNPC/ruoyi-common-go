// Package strutil 字符串工具。
package strutil

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Capitalize 将 s 首字母转为大写，其余字符不变，返回转换后的字符串。
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	upper := unicode.ToUpper(r)
	if upper == r {
		return s
	}
	return string(upper) + s[size:]
}

// IsHTTP 判断 link 是否以 http:// 或 https:// 开头。
func IsHTTP(link string) bool {
	return strings.HasPrefix(strings.ToLower(link), "http://") ||
		strings.HasPrefix(strings.ToLower(link), "https://")
}

// ParseIDList 将逗号分隔的主键串 raw 解析为 int64 切片返回；
// 任一段非整数或 ≤0 即返回 error。
func ParseIDList(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("strutil: 非法主键 %q", p)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
