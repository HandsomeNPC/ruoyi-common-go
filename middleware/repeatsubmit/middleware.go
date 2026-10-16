package repeatsubmit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	sagin "github.com/HandsomeNPC/sa-token-go/integrations/gin"
	"github.com/gin-gonic/gin"
)

// run 执行一次防重判定。
func (s *Submitter) run(c *gin.Context, o *options) {
	key := s.combineKey(c)

	locked, err := s.acquire(c.Request.Context(), key, o.interval)
	if err != nil {
		// Redis 异常时放行：可用性优先，异常已记日志。
		log.Printf("[repeatsubmit] %s %s 防重判定异常,已放行: %v",
			c.Request.Method, c.Request.URL.Path, err)
		c.Next()
		return
	}
	if !locked {
		// 直接写失败响应：原先登记 c.Error 交给 Recover 渲染，但该中间件未必被
		// Recover 外包（即便外包，写入也会被本函数后续缓冲的 bodyWriter 吞掉），
		// 客户端拿不到提示。自己渲染才稳定。
		log.Printf("[repeatsubmit] %s %s 触发防重提交, 缓存key => '%s'",
			c.Request.Method, c.Request.URL.Path, key)
		c.AbortWithStatusJSON(http.StatusOK, failBody{Code: codeFail, Msg: o.message})
		return
	}

	// 缓冲响应体：要读到业务 code 才能判定成功与否。
	original := c.Writer
	buf := &bodyWriter{ResponseWriter: original}
	c.Writer = buf

	completed := false
	defer func() {
		c.Writer = original
		if completed {
			return
		}
		// panic 正在展开：删键放行重试。缓冲区丢弃不 flush，
		// 否则 c.Writer.Written() 置 true，Recover 不再渲染 500，客户端只收到半截响应。
		s.release(c, key)
	}()

	c.Next()
	completed = true

	if s.succeeded(c, buf) {
		buf.flush() // 成功：保留键，interval 内挡住重复提交。
		return
	}
	s.release(c, key) // 失败：删键，允许立刻重试。
	buf.flush()
}

// succeeded 判定本次请求是否成功：c.Error 非空、HTTP 非 2xx、响应 code 非 200 都算失败。
// 响应体不是 R 结构（文件流、空响应等）时按成功处理（保留键）。
func (s *Submitter) succeeded(c *gin.Context, buf *bodyWriter) bool {
	if len(c.Errors) > 0 {
		return false
	}
	if status := c.Writer.Status(); status < 200 || status > 299 {
		return false
	}
	code, ok := parseCode(buf.body.Bytes())
	if !ok {
		return true
	}
	return code == codeSuccess
}

// acquire 抢占防重键，返回 true 表示抢到（放行）。
func (s *Submitter) acquire(ctx context.Context, key string, interval time.Duration) (bool, error) {
	return s.rdb.SetNX(ctx, key, "", interval).Result()
}

// release 删除防重键。
func (s *Submitter) release(c *gin.Context, key string) {
	// 用 WithoutCancel：客户端断连会取消请求 context，但键仍要删，否则重连后白等一个 interval。
	ctx := context.WithoutCancel(c.Request.Context())
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		log.Printf("[repeatsubmit] %s %s 删除防重键失败, 缓存key => '%s': %v",
			c.Request.Method, c.Request.URL.Path, key, err)
	}
}

// combineKey 组装防重缓存键：global:repeat_submit:<请求路径><sha256(token+":"+入参)>。
func (s *Submitter) combineKey(c *gin.Context) string {
	token := s.token(c)

	sum := sha256.New()
	sum.Write([]byte(token))
	sum.Write([]byte(":"))
	sum.Write(requestParams(c))
	fingerprint := hex.EncodeToString(sum.Sum(nil))

	var sb strings.Builder
	sb.WriteString(repeatSubmitKey)
	sb.WriteString(c.Request.URL.Path)
	sb.WriteString(fingerprint)
	return sb.String()
}

// token 取当前请求的 token 作为防重用户维度，优先取 sagin 写入 ctx 的值，回落到读请求头。
func (s *Submitter) token(c *gin.Context) string {
	if t := strings.TrimSpace(sagin.GetTokenFromCtx(c)); t != "" {
		return t
	}
	return strings.TrimSpace(c.GetHeader(s.tokenName))
}

// requestParams 取参与指纹的入参（请求体 + query 串）。
// 二者都无时指纹仅由 token 与路径决定。
func requestParams(c *gin.Context) []byte {
	query := c.Request.URL.RawQuery
	body := requestBody(c)

	if query == "" {
		return body
	}
	if len(body) == 0 {
		return []byte(query)
	}
	buf := make([]byte, 0, len(body)+len(query)+1)
	buf = append(buf, body...)
	buf = append(buf, ' ')
	buf = append(buf, query...)
	return buf
}

// requestBody 取请求体，优先用已缓存的请求体副本；未缓存时读后塞回 Body 保证 handler 仍能读到。
func requestBody(c *gin.Context) []byte {
	v, _ := c.Get(gin.BodyBytesKey)
	body, _ := v.([]byte)
	if body != nil {
		return body
	}
	if c.Request.Body == nil {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxFingerprintBody))
	if err != nil {
		log.Printf("[repeatsubmit] %s %s 读取请求体失败,指纹降级为仅按路径与 token 计算: %v",
			c.Request.Method, c.Request.URL.Path, err)
		return nil
	}
	// 读完塞回，超过上限的部分仍留在原 Body 里，用 MultiReader 接回。
	c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), c.Request.Body))
	return body
}

// parseCode 从响应体取业务 code，返回 ok=false 表示不是 R 结构（按成功处理）。只解 code 字段。
func parseCode(body []byte) (int, bool) {
	if len(body) == 0 {
		return 0, false
	}
	var r struct {
		Code *int `json:"code"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.Code == nil {
		return 0, false
	}
	return *r.Code, true
}

// bodyWriter 缓冲响应体的 gin.ResponseWriter 包装。
type bodyWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error)       { return w.body.Write(b) }
func (w *bodyWriter) WriteString(s string) (int, error) { return w.body.WriteString(s) }

func (w *bodyWriter) flush() {
	if w.body.Len() == 0 {
		return // handler 没写响应（可能走了 c.Error 由 Recover 渲染）
	}
	_, _ = w.ResponseWriter.Write(w.body.Bytes())
}
