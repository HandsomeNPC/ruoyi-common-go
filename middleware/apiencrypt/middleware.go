// Package apiencrypt 接口加解密中间件。
//
// RSA 协商 AES 密钥、AES 加解密报文体：POST/PUT 命中注解且带密钥头则用配置的 RSA 私钥解出
// AES 密钥并解密请求体，可选再用公钥随机生成 AES 密钥加密响应体。!Enabled 时中间件直通。
//
// 自包含、不引 ruoyi-core/middleware 与 response：失败用 fmt 直写 {code,msg,data} JSON，
// 不走 errs + 全局 Recover 兜底，免去 errs 依赖与「Abort 后等 Recover 才落响应」的隐式时序。
// 响应码与原 errs 路径一致：缺密钥头 403，解密/加密失败回落 500。
package apiencrypt

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultHeader  = "encrypt-key"    // 默认密钥头名
	defaultMaxBody = 10 * 1024 * 1024 // 默认密文上限 10MB
)

// Crypto 接口加解密器，持有解析后的密钥。!Enabled 时为零值，run 直通。
type Crypto struct {
	header  string
	priv    *rsa.PrivateKey
	pub     *rsa.PublicKey
	maxSize int64
}

// newCrypto 解析配置构造 Crypto。!Enabled 返回零值（中间件直通）；密钥解析失败 panic
// （启动期 fail-fast，与原 encrypt.Init 行为一致）。
func newCrypto(cfg APIEncryptConfig) *Crypto {
	if !cfg.Enabled {
		return &Crypto{}
	}
	priv, err := ParseRSAPrivateKey(cfg.PrivateKey)
	if err != nil {
		panic(fmt.Errorf("apiencrypt: 初始化失败: %w", err))
	}
	var pub *rsa.PublicKey
	if cfg.PublicKey != "" {
		pub, err = ParseRSAPublicKey(cfg.PublicKey)
		if err != nil {
			panic(fmt.Errorf("apiencrypt: 初始化失败: %w", err))
		}
	}
	header := cfg.HeaderFlag
	if header == "" {
		header = defaultHeader
	}
	return &Crypto{header: header, priv: priv, pub: pub, maxSize: defaultMaxBody}
}

// APIEncryptWithConfig 请求解密中间件：POST/PUT 命中注解则要求带密钥头并解密请求体，未带则 403。
func APIEncryptWithConfig(cfg APIEncryptConfig) gin.HandlerFunc {
	e := newCrypto(cfg)
	return func(c *gin.Context) { e.run(c, false) }
}

// APIEncryptWithResponseWithConfig 请求解密 + 响应加密中间件。
func APIEncryptWithResponseWithConfig(cfg APIEncryptConfig) gin.HandlerFunc {
	e := newCrypto(cfg)
	return func(c *gin.Context) { e.run(c, true) }
}

// run 执行加解密。
func (e *Crypto) run(c *gin.Context, withResponse bool) {
	if e.priv == nil {
		// 未启用（!Enabled）：中间件直通，不做任何加解密。
		c.Next()
		return
	}
	if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
		headerValue := strings.TrimSpace(c.GetHeader(e.header))
		if headerValue == "" {
			// 命中加密注解却未带密钥头 → 403。
			log.Printf("[apiencrypt] %s %s 要求加密传输但未携带 %s 头",
				c.Request.Method, c.Request.URL.Path, e.header)
			writeFail(c, 403, "没有访问权限，请联系管理员授权")
			return
		}
		if !e.decryptRequest(c, headerValue) {
			return
		}
	}
	if !withResponse {
		c.Next()
		return
	}
	e.encryptResponse(c)
}

// decryptRequest 解密请求体并换回明文 body。
func (e *Crypto) decryptRequest(c *gin.Context, headerValue string) bool {
	if c.Request.ContentLength > e.maxSize {
		e.fail(c, "请求体超限")
		return false
	}
	if c.Request.Body == nil {
		e.fail(c, "请求体为空")
		return false
	}
	cipherBody, err := io.ReadAll(io.LimitReader(c.Request.Body, e.maxSize+1))
	if err != nil {
		e.fail(c, "读取请求体失败: "+err.Error())
		return false
	}
	if int64(len(cipherBody)) > e.maxSize {
		e.fail(c, "请求体超限")
		return false
	}
	// RSA 私钥解出 base64 编码的 AES 密钥。
	decryptedHeader, err := DecryptByRSA(headerValue, e.priv)
	if err != nil {
		e.fail(c, "RSA 解密 AES 秘钥失败: "+err.Error())
		return false
	}
	// 再 base64 解一次才是真正的 AES 密钥。
	aesPassword, err := DecryptByBase64(decryptedHeader)
	if err != nil {
		e.fail(c, "AES 秘钥 base64 解码失败: "+err.Error())
		return false
	}
	// AES 解密请求体。
	plain, err := DecryptByAES(unwrapJSONString(bytes.TrimSpace(cipherBody)), aesPassword)
	if err != nil {
		e.fail(c, "AES 解密请求体失败: "+err.Error())
		return false
	}
	body := []byte(plain)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// 同步更新 BodyBytesKey：RepeatableBody 在本中间件之前已缓存密文，此处覆盖为明文，
	// 让下游 c.Get(gin.BodyBytesKey) 取到的是明文。
	c.Set(gin.BodyBytesKey, body)
	return true
}

// encryptResponse 缓冲响应体，结束后整体加密再写出。
func (e *Crypto) encryptResponse(c *gin.Context) {
	if e.pub == nil {
		log.Printf("[apiencrypt] %s 配置了响应加密但未提供公钥", c.Request.URL.Path)
		writeFail(c, 500, "响应加密失败")
		return
	}
	aesPassword, err := GenerateAESPassword()
	if err != nil {
		e.failResponse(c, "生成 AES 秘钥失败: "+err.Error())
		return
	}
	encryptedKey, err := EncryptByRSA(EncryptByBase64(aesPassword), e.pub)
	if err != nil {
		e.failResponse(c, "RSA 加密 AES 秘钥失败: "+err.Error())
		return
	}
	// 密钥头必须在 c.Next() 之前写，body 一开始输出头就发出去了。
	c.Writer.Header().Set(e.header, encryptedKey)
	c.Writer.Header().Add("Access-Control-Expose-Headers", e.header)

	original := c.Writer
	buf := &cryptoWriter{ResponseWriter: original}
	c.Writer = buf
	c.Next()
	// 还原 c.Writer，否则 Recover 在本函数返回后渲染错误会落进已用完的缓冲区。
	c.Writer = original

	if buf.body.Len() == 0 {
		// handler 没写响应（可能走了 c.Error 由 Recover 渲染），撤掉密钥头。
		c.Writer.Header().Del(e.header)
		return
	}
	cipherText, err := EncryptByAES(buf.body.String(), aesPassword)
	if err != nil {
		// 加密失败绝不能退回写明文，丢弃响应体并回错误。
		log.Printf("[apiencrypt] %s 响应加密失败,已丢弃响应体: %v", c.Request.URL.Path, err)
		buf.discardAndFail(e.header)
		return
	}
	buf.writeEncrypted(cipherText)
}

// unwrapJSONString 剥掉密文外层的 JSON 字符串引号。
//
// 前端 axios 把加密后的 base64 串当 JSON body 发（Content-Type 是 application/json），
// transformRequest 会 JSON.stringify 一次，于是上线的 body 是 "abc123=="——带引号。
// 原实现用 hutool 的 base64 解码器会静默跳过非字母表字符（引号被忽略），
// 而 Go 标准库 base64 严格报错(illegal base64 data at input byte 0)，故须显式剥离。
func unwrapJSONString(b []byte) string {
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		return string(b[1 : len(b)-1])
	}
	return string(b)
}

// writeFail 直写失败 JSON 并终止链：HTTP 恒 200，body code/msg 由参数给。
// 不定义 envelope 类型、不引 ruoyi-core/response，按项目统一 {code,msg,data} 形态用 fmt 拼。
func writeFail(c *gin.Context, code int, msg string) {
	c.Data(http.StatusOK, "application/json; charset=utf-8",
		[]byte(fmt.Sprintf(`{"code":%d,"msg":"%s","data":null}`, code, msg)))
	c.Abort()
}

// fail 记录明细并回请求解密失败的统一文案。响应码与原 errs.New(0,...) 经 Recover 回落 500 一致。
func (e *Crypto) fail(c *gin.Context, detail string) {
	log.Printf("[apiencrypt] %s %s 解密失败: %s", c.Request.Method, c.Request.URL.Path, detail)
	writeFail(c, 500, "请求解密失败")
}

// failResponse 响应加密前置步骤失败时回错误，此时 handler 还没执行。
func (e *Crypto) failResponse(c *gin.Context, detail string) {
	log.Printf("[apiencrypt] %s 响应加密准备失败: %s", c.Request.URL.Path, detail)
	writeFail(c, 500, "响应加密失败")
}

// cryptoWriter 缓冲响应体的 gin.ResponseWriter 包装。
type cryptoWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *cryptoWriter) Write(b []byte) (int, error)       { return w.body.Write(b) }
func (w *cryptoWriter) WriteString(s string) (int, error) { return w.body.WriteString(s) }

// writeEncrypted 把密文写进真正的响应。
func (w *cryptoWriter) writeEncrypted(cipherText string) {
	h := w.ResponseWriter.Header()
	h.Set("Content-Length", strconv.Itoa(len(cipherText))) // 长度按密文重算
	h.Set("Content-Type", "text/plain;charset=utf-8")      // 密文是 base64 纯文本
	_, _ = w.ResponseWriter.WriteString(cipherText)
}

// discardAndFail 丢弃已缓冲的明文并写一个加密失败的响应。
func (w *cryptoWriter) discardAndFail(header string) {
	w.body.Reset()
	h := w.ResponseWriter.Header()
	h.Del(header) // 密钥头必须撤掉：响应体已不是用它加密的
	h.Set("Content-Type", "application/json; charset=utf-8")
	body := fmt.Sprintf(`{"code":%d,"msg":"%s","data":null}`, 500, "响应加密失败")
	h.Set("Content-Length", strconv.Itoa(len(body)))
	w.ResponseWriter.WriteHeader(http.StatusOK)
	_, _ = w.ResponseWriter.Write([]byte(body))
}
