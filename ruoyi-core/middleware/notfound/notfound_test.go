package notfound

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 404/405 必须是 HTTP 200 + R，与 Java GlobalExceptionHandler 的两个分支对齐。
func TestFallbacks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/exists", func(c *gin.Context) { c.JSON(200, gin.H{"ok": 1}) })
	Register(r)

	cases := []struct {
		name     string
		method   string
		path     string
		wantCode float64
		wantMsg  string
	}{
		{"未注册路径 -> 404", http.MethodGet, "/nope", 404, msgNotFound},
		{"错动词 -> 405", http.MethodPost, "/exists", 405, "Request method 'POST' is not supported"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))

			if w.Code != http.StatusOK {
				t.Fatalf("HTTP 状态必须为 200(前端才会读 code), 实际 %d body=%s", w.Code, w.Body.String())
			}
			var got map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("响应非 JSON: %v body=%s", err, w.Body.String())
			}
			if got["code"] != tc.wantCode {
				t.Errorf("code 应为 %v, 实际 %v", tc.wantCode, got["code"])
			}
			if got["msg"] != tc.wantMsg {
				t.Errorf("msg 应为 %q, 实际 %v", tc.wantMsg, got["msg"])
			}
			if _, ok := got["data"]; !ok {
				t.Error("data 字段必须存在(序列化为 null)")
			}
			t.Logf("%s %s -> HTTP %d %s", tc.method, tc.path, w.Code, w.Body.String())
		})
	}
}

// 正常路由不受兜底影响。
func TestNormalRouteUnaffected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/exists", func(c *gin.Context) { c.JSON(200, gin.H{"ok": 1}) })
	Register(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/exists", nil))
	if w.Code != 200 || w.Body.String() != `{"ok":1}` {
		t.Errorf("正常路由被影响: HTTP %d body=%s", w.Code, w.Body.String())
	}
}
