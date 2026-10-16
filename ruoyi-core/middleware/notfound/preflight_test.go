package notfound

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 回归防护：开启 HandleMethodNotAllowed 后，CORS 预检(OPTIONS)不能被 405 拦掉。
// 全局中间件会与 NoRoute/NoMethod 的处理链合并，故先 Abort 的 CORS 应当赢。
func TestPreflightNotSwallowedBy405(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 模拟 cors 中间件：预检直接 200 + Abort
	r.Use(func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})
	r.GET("/exists", func(c *gin.Context) { c.JSON(200, gin.H{"ok": 1}) })
	Register(r)

	req := httptest.NewRequest(http.MethodOptions, "/exists", nil)
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	t.Logf("OPTIONS /exists -> HTTP %d body=%q", w.Code, w.Body.String())
	if w.Code != http.StatusOK {
		t.Fatalf("预检应被 CORS 以 200 放过, 实际 %d —— HandleMethodNotAllowed 吞掉了预检", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("预检响应不应有 body, 实际 %q", w.Body.String())
	}
}
