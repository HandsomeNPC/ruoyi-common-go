package accesslog

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 敏感字段脱敏必须忽略大小写：JSON 绑定本身不区分大小写（{"Password":...} 照样
// 能绑进 Password 字段），逐字匹配会让 "Password"/"PASSWORD" 明文落进访问日志。
func TestSanitizeJSONParamIsCaseInsensitive(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"小写", `{"userName":"admin","password":"Secret123!"}`},
		{"首字母大写", `{"userName":"admin","Password":"Secret123!"}`},
		{"全大写", `{"userName":"admin","PASSWORD":"Secret123!"}`},
		{"驼峰变体", `{"userName":"admin","OldPassword":"Secret123!"}`},
		{"嵌套对象", `{"user":{"NewPassword":"Secret123!"}}`},
		{"数组内嵌", `{"list":[{"ConfirmPassword":"Secret123!"}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeJSONParam([]byte(tt.body), 4096, nil)
			if strings.Contains(got, "Secret123!") {
				t.Errorf("密码明文泄漏进日志: %s", got)
			}
			if !strings.Contains(strings.ToLower(got), "username") &&
				!strings.Contains(got, "user") && !strings.Contains(got, "list") {
				t.Errorf("非敏感字段被误删: %s", got)
			}
		})
	}
}

func TestSanitizeFormParamIsCaseInsensitive(t *testing.T) {
	for _, key := range []string{"password", "Password", "PASSWORD", "oldPassword", "OLDPASSWORD"} {
		t.Run(key, func(t *testing.T) {
			form := url.Values{key: {"Secret123!"}, "userName": {"admin"}}
			req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req

			got := sanitizeFormParam(c, 4096, nil)
			if strings.Contains(got, "Secret123!") {
				t.Errorf("密码明文泄漏进日志: %s", got)
			}
			if !strings.Contains(got, "admin") {
				t.Errorf("非敏感字段被误删: %s", got)
			}
		})
	}
}

// 脱敏只应作用于日志副本，不能改动 handler 后面要读的请求数据。
func TestSanitizeFormParamDoesNotMutateRequest(t *testing.T) {
	form := url.Values{"Password": {"Secret123!"}, "userName": {"admin"}}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	_ = sanitizeFormParam(c, 4096, nil)

	if got := c.Request.Form.Get("Password"); got != "Secret123!" {
		t.Errorf("请求体被脱敏逻辑改动，handler 将读不到原值: Password=%q", got)
	}
}

// 额外追加的敏感字段同样忽略大小写。
func TestExtraExcludeFieldsAreCaseInsensitive(t *testing.T) {
	got := sanitizeJSONParam([]byte(`{"Token":"abc123","keep":"ok"}`), 4096, []string{"token"})
	if strings.Contains(got, "abc123") {
		t.Errorf("调用方追加的敏感字段未脱敏: %s", got)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("脱敏结果不是合法 JSON: %v (%s)", err, got)
	}
	if m["keep"] != "ok" {
		t.Errorf("非敏感字段被误删: %s", got)
	}
}
