package operlog

import (
	"bytes"
	"strings"

	"github.com/gin-gonic/gin"
)

// responseCapture 抄一份响应体的 gin.ResponseWriter 包装。
// 写入原样透传，另抄一份供落库；抄本封顶，导出接口的 xlsx 不该整个堆在内存里。
type responseCapture struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

// maxCaptureBytes 抄本上限。落库截到 maxContentLength 个字符，按 UTF-8 单字符最多 4 字节留够余量。
const maxCaptureBytes = maxContentLength * 4

func (w *responseCapture) Write(b []byte) (int, error) {
	w.capture(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseCapture) WriteString(s string) (int, error) {
	w.capture([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

// capture 把响应字节抄进缓冲区，超上限丢弃。先看 Content-Type：非 JSON 响应最终会被
// text 丢掉，白抄一遍给每次导出多分配一份内存。
func (w *responseCapture) capture(b []byte) {
	if !w.jsonBody() {
		return
	}
	if room := maxCaptureBytes - w.buf.Len(); room > 0 {
		w.buf.Write(b[:min(len(b), room)])
	}
}

func (w *responseCapture) jsonBody() bool {
	return strings.HasPrefix(
		strings.ToLower(w.Header().Get("Content-Type")), contentTypeJSON)
}

// text 返回抄本。非 JSON 响应返回空串：导出接口的响应体是 xlsx 二进制，
// 落进 json_result 既无可读性又白占 4000 字节。
func (w *responseCapture) text() string {
	if !w.jsonBody() {
		return ""
	}
	return w.buf.String()
}
