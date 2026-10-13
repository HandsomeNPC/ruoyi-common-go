package operlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/constant"

	"github.com/HandsomeNPC/ruoyi-common-go/jsonx"
)

// 请求参数脱敏与截断。本包自带一份，与 accesslog 各自维护——换取本包完全自包含，
// 不引外部脱敏包。敏感字段来源仍是 constant.ExcludeProperties。

// contentTypeJSON JSON 请求/响应的 content-type 前缀。
const contentTypeJSON = "application/json"

// maxSanitizeSize 结构化脱敏的体积上限，超过走原文兜底路径。
const maxSanitizeSize = 256 << 10

// msgSensitiveParamOmitted 参数无法结构化脱敏且疑似含敏感字段时的占位文案。
const msgSensitiveParamOmitted = "<参数无法解析且疑似含敏感字段，已省略>"

// requestParam 采集入参。query/form 有值就用它，否则 PUT/POST/DELETE 取 body+路径参数。
// 带路径参数：否则 DELETE /client/{ids} 只剩一条空记录，而删了哪几个客户端恰是审计所需。
func requestParam(c *gin.Context, exclude []string) string {
	if params := sanitizeFormParam(c, maxContentLength, exclude); params != "" {
		return params
	}
	switch c.Request.Method {
	case http.MethodPost, http.MethodPut, http.MethodDelete:
		v, _ := c.Get(gin.BodyBytesKey)
		body, _ := v.([]byte)
		return joinParams(
			pathParam(c, exclude),
			sanitizeJSONParam(body, maxContentLength, exclude),
		)
	default:
		return ""
	}
}

// pathParam 把路径参数序列化成 JSON 串，无参数返回空串。
func pathParam(c *gin.Context, exclude []string) string {
	if len(c.Params) == 0 {
		return ""
	}
	params := make(map[string]string, len(c.Params))
	for _, p := range c.Params {
		if !excluded(p.Key, exclude) {
			params[p.Key] = p.Value
		}
	}
	if len(params) == 0 {
		return ""
	}
	out, err := jsonx.Marshal(params)
	if err != nil {
		return ""
	}
	return string(out)
}

// excluded 判断参数名是否脱敏，比对不区分大小写。
func excluded(name string, extra []string) bool {
	return isSensitive(name, excludeFields(extra))
}

// isSensitive 判断字段名是否属于脱敏集，比对不区分大小写。
// 大小写不敏感是脱敏一致性的底线：JSON key 大小写由前端决定（"Password" 与 "password"），
// 精确匹配会漏脱，路径参数与原文兜底已是不敏感，这里对齐。
func isSensitive(name string, fields []string) bool {
	for _, field := range fields {
		if strings.EqualFold(name, field) {
			return true
		}
	}
	return false
}

// joinParams 用空格拼接非空片段。
func joinParams(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return limit(strings.Join(kept, " "), maxContentLength)
}

// sanitizeJSONParam 把 JSON 请求体脱敏并截断成一行文本，空体返回空串。
// exclude 是在 constant.ExcludeProperties 之外额外要剔除的字段名。
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
		return rawParamLog(body, maxLen, fields) // 解析失败不影响主请求，退回原文。
	}

	removeSensitiveFields(node, fields)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // 关掉 HTML 转义，日志是给人看的。
	if err := enc.Encode(node); err != nil {
		return rawParamLog(body, maxLen, fields)
	}

	// Encode 会补一个换行符，留着会把一行日志撑成两行。
	return limit(strings.TrimRight(buf.String(), "\n"), maxLen)
}

// sanitizeFormParam 把 query 与表单参数脱敏并截断成一行文本，无参数时返回空串。
func sanitizeFormParam(c *gin.Context, maxLen int, exclude []string) string {
	_ = c.Request.ParseForm()

	form := c.Request.Form
	if len(form) == 0 {
		return ""
	}

	// 复制一份再删：Form 是 handler 后面要用的数据，日志绝不能改动请求本身。
	safe := make(url.Values, len(form))
	for k, v := range form {
		safe[k] = v
	}
	for _, field := range excludeFields(exclude) {
		delete(safe, field)
	}
	if len(safe) == 0 {
		return ""
	}

	// 序列化的是 map[string][]string，同名参数可出现多次。
	out, err := json.Marshal(safe)
	if err != nil {
		return ""
	}
	return limit(string(out), maxLen)
}

// excludeFields 合并全局敏感字段与调用方追加的字段。
func excludeFields(extra []string) []string {
	if len(extra) == 0 {
		return constant.ExcludeProperties
	}
	fields := make([]string, 0, len(constant.ExcludeProperties)+len(extra))
	fields = append(fields, constant.ExcludeProperties...)
	return append(fields, extra...)
}

// removeSensitiveFields 递归删掉敏感字段，键名比对不区分大小写。
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

// rawParamLog 结构化脱敏失败时的兜底：截断原文，但先确认里面没有敏感字段。
func rawParamLog(body []byte, maxLen int, fields []string) string {
	raw := limit(string(body), maxLen)
	lower := strings.ToLower(raw)
	for _, field := range fields {
		if strings.Contains(lower, strings.ToLower(field)) {
			return msgSensitiveParamOmitted
		}
	}
	return raw
}
