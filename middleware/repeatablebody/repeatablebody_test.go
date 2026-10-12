package repeatablebody

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// newEngine 装一个只回显缓存 body 的测试路由。
func newEngine(cfg RepeatableBodyConfig) *gin.Engine {
	r := gin.New()
	r.Use(RepeatableBodyWithConfig(cfg))
	r.POST("/echo", func(c *gin.Context) {
		v, _ := c.Get(gin.BodyBytesKey)
		body, _ := v.([]byte)
		c.Data(http.StatusOK, "text/plain", body)
	})
	return r
}

// TestDefaultConfig 全为零值时应回落到默认值。
func TestDefaultConfig(t *testing.T) {
	var d RepeatableBodyConfig
	d.applyDefaults()

	if got, want := d.MaxBodySize, int64(defaultMaxBodySize); got != want {
		t.Errorf("默认 MaxBodySize = %d, want %d", got, want)
	}
	if len(d.ContentTypes) != 1 || d.ContentTypes[0] != contentTypeJSON {
		t.Errorf("默认 ContentTypes = %v, want [%q]", d.ContentTypes, contentTypeJSON)
	}
	if err := d.validate(); err != nil {
		t.Errorf("默认配置应校验通过, got %v", err)
	}
	if err := (RepeatableBodyConfig{MaxBodySize: -1}).validate(); err == nil {
		t.Error("负 MaxBodySize 应校验失败")
	}
}

// TestJSONBodyCachedAndReReadable JSON body 应被缓存且 handler 能读到完整内容。
func TestJSONBodyCachedAndReReadable(t *testing.T) {
	r := newEngine(RepeatableBodyConfig{ContentTypes: []string{contentTypeJSON}, MaxBodySize: 1 << 20})

	body := `{"name":"ruoyi"}`
	req := httptest.NewRequest("POST", "/echo", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200", w.Code)
	}
	if w.Body.String() != body {
		t.Errorf("回显 body = %q, want %q", w.Body.String(), body)
	}
}

// TestOversizedRejected 超出上限的请求体应被直接拒绝并回写统一失败响应。
func TestOversizedRejected(t *testing.T) {
	r := newEngine(RepeatableBodyConfig{ContentTypes: []string{contentTypeJSON}, MaxBodySize: 4})

	req := httptest.NewRequest("POST", "/echo", strings.NewReader("hello"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "请求体超出大小限制") {
		t.Errorf("响应应含超限提示, got %q", w.Body.String())
	}
}

// TestNonListedContentTypeSkipped 未声明的 content-type 不缓存，BodyBytes 为空。
func TestNonListedContentTypeSkipped(t *testing.T) {
	r := newEngine(RepeatableBodyConfig{ContentTypes: []string{contentTypeJSON}, MaxBodySize: 1 << 20})

	req := httptest.NewRequest("POST", "/echo", strings.NewReader("plain"))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200", w.Code)
	}
	if w.Body.String() != "" {
		t.Errorf("未缓存时回显应为空, got %q", w.Body.String())
	}
}

// TestShouldBufferBody 无请求体与未命中 content-type 都不应缓存。
func TestShouldBufferBody(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	types := []string{contentTypeJSON}

	// GET 无请求体：ContentLength==0，不缓存。
	c.Request = httptest.NewRequest("GET", "/", nil)
	if shouldBufferBody(c, types) {
		t.Error("无请求体时不应缓存")
	}

	// 命中 content-type 的 POST：缓存。
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader("{}"))
	c.Request.Header.Set("Content-Type", "application/json")
	if !shouldBufferBody(c, types) {
		t.Error("命中 content-type 应缓存")
	}

	// 未命中 content-type：不缓存。
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader("plain"))
	c.Request.Header.Set("Content-Type", "text/plain")
	if shouldBufferBody(c, types) {
		t.Error("未命中 content-type 不应缓存")
	}
}
