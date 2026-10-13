package accesslog

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// 请求参数脱敏与截断。本包自带一份，与 oplog 各自维护——accesslog 不引外部脱敏包，
// 换取中间件包的完全自包含。

// excludeProperties 敏感字段集。与 operlog 那份是故意的两份副本（换取本包自包含），
// 改这里时必须同步 ruoyi-core/middleware/operlog/param.go——两边曾因此分叉过：
// 一侧逐字匹配、一侧忽略大小写，导致 {"Password":...} 明文进了访问日志。
// 行为约束见 param_test.go。
var excludeProperties = []string{"password", "oldPassword", "newPassword", "confirmPassword"}

// contentTypeJSON JSON 请求的 content-type 前缀。
const contentTypeJSON = "application/json"

// truncatedSuffix 截断标记。
const truncatedSuffix = "...(已截断)"

// maxSanitizeSize 结构化脱敏的体积上限，超过走原文兜底路径。
const maxSanitizeSize = 256 << 10

// msgSensitiveParamOmitted 参数无法结构化脱敏且疑似含敏感字段时的占位文案。
const msgSensitiveParamOmitted = "<参数无法解析且疑似含敏感字段，已省略>"

// sanitizeJSONParam 把 JSON 请求体脱敏并截断成一行文本，空体返回空串。
// exclude 是在 excludeProperties 之外额外要剔除的字段名。
func sanitizeJSONParam(body []byte, maxLen int, exclude []string) string {
	if len(body) == 0 {
		return ""
	}
	fields := excludeFields(exclude)
	if len(body) > maxSanitizeSize {
		return rawParamLog(body, maxLen, fields)
	}

	// UseNumber 不能省：默认把数字解成 float64，会抹掉雪花 id 的尾数。
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var node any
	if err := dec.Decode(&node); err != nil {
		// 解析失败不影响主请求，退回原文。
		return rawParamLog(body, maxLen, fields)
	}

	removeSensitiveFields(node, fields)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// 关掉 HTML 转义，日志是给人看的。
	enc.SetEscapeHTML(false)
	if err := enc.Encode(node); err != nil {
		return rawParamLog(body, maxLen, fields)
	}

	// Encode 会补一个换行符，留着会把一行日志撑成两行。
	return limitParam(strings.TrimRight(buf.String(), "\n"), maxLen)
}

// sanitizeFormParam 把 query 与表单参数脱敏并截断成一行文本，无参数时返回空串。
func sanitizeFormParam(c *gin.Context, maxLen int, exclude []string) string {
	_ = c.Request.ParseForm()

	form := c.Request.Form
	if len(form) == 0 {
		return ""
	}

	// 复制一份再删：Form 是 handler 后面要用的数据，日志绝不能改动请求本身。
	fields := excludeFields(exclude)
	safe := make(url.Values, len(form))
	for k, v := range form {
		if isSensitive(k, fields) {
			continue
		}
		safe[k] = v
	}
	if len(safe) == 0 {
		return ""
	}

	// 序列化的是 map<string, []string>，同名参数可出现多次。
	out, err := json.Marshal(safe)
	if err != nil {
		return ""
	}
	return limitParam(string(out), maxLen)
}

// excludeFields 合并全局敏感字段与调用方追加的字段。
func excludeFields(extra []string) []string {
	if len(extra) == 0 {
		return excludeProperties
	}
	fields := make([]string, 0, len(excludeProperties)+len(extra))
	fields = append(fields, excludeProperties...)
	return append(fields, extra...)
}

// removeSensitiveFields 递归删掉敏感字段。
//
// 按忽略大小写匹配：JSON 绑定本身不区分大小写（{"Password":...} 同样能绑进
// Password 字段），逐字匹配会漏掉 "Password"/"PASSWORD" 这类写法导致明文入日志。
func removeSensitiveFields(node any, fields []string) {
	switch n := node.(type) {
	case map[string]any:
		// Go 规范允许遍历 map 时删除当前 key，命中的就地删掉再递归剩下的子节点。
		for key := range n {
			if isSensitive(key, fields) {
				delete(n, key)
			}
		}
		for _, child := range n {
			removeSensitiveFields(child, fields)
		}
	case []any:
		for _, child := range n {
			removeSensitiveFields(child, fields)
		}
	}
}

// isSensitive 判断字段名是否命中敏感字段集，忽略大小写。
func isSensitive(name string, fields []string) bool {
	for _, field := range fields {
		if strings.EqualFold(name, field) {
			return true
		}
	}
	return false
}

// rawParamLog 结构化脱敏失败时的兜底：截断原文，但先确认里面没有敏感字段。
func rawParamLog(body []byte, maxLen int, fields []string) string {
	raw := limitParam(string(body), maxLen)
	lower := strings.ToLower(raw)
	for _, field := range fields {
		if strings.Contains(lower, strings.ToLower(field)) {
			return msgSensitiveParamOmitted
		}
	}
	return raw
}

// limitParam 按字符数截断参数日志。
func limitParam(s string, maxLen int) string {
	// 一个 rune 至少占 1 字节，字节数没超就一定没超字符数。
	if len(s) <= maxLen {
		return s
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}

	count := 0
	for i := range s {
		if count == maxLen {
			return s[:i] + truncatedSuffix
		}
		count++
	}
	return s + truncatedSuffix
}
