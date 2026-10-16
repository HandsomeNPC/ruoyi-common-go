package checklogin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/middleware/recover"
)

// 未登录(无 token)时必须是 HTTP 200 + {"code":401,...}，与 Java R.fail(401,...) 对齐。
func TestCheckLoginNoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(recover.Recover())
	r.GET("/t", CheckLogin(), func(c *gin.Context) { c.String(200, "reached") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态应为 200(前端 axios 才会走 code==401 跳登录页), 实际 %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应非 JSON: %v, body=%s", err, w.Body.String())
	}
	if got := body["code"]; got != float64(401) {
		t.Errorf("code 应为 401, 实际 %v", got)
	}
	if _, ok := body["msg"]; !ok {
		t.Errorf("应有 msg 字段(项目契约 code/msg/data), 实际 body=%s", w.Body.String())
	}
	if _, ok := body["message"]; ok {
		t.Errorf("不应出现 sa-token 的 message 字段, body=%s", w.Body.String())
	}
	if _, ok := body["error"]; ok {
		t.Errorf("不应泄漏原始 error 字段, body=%s", w.Body.String())
	}
	if body["msg"] != msgDefault {
		t.Errorf("文案应为 %q, 实际 %v", msgDefault, body["msg"])
	}
	t.Logf("响应: HTTP %d %s", w.Code, w.Body.String())
}
