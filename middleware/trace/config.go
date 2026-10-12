package trace

// TraceIDHeader 读写链路 id 的请求/响应头名。
const TraceIDHeader = "X-Request-Id"

// TraceIDConfig 链路 id 中间件配置。
type TraceIDConfig struct {
	// Header 读写链路 id 的头名，默认 TraceIDHeader。
	Header string `mapstructure:"header"`

	// TrustInbound 是否沿用入站请求头里已有的 id。
	TrustInbound bool `mapstructure:"trustInbound"`
}

// applyDefaults 就地铺兜底默认值。Header 为空视为未配置，一并补 Header 与 TrustInbound=true
// （bool 无零值可辨「未设」，故仅在整体未配置时兜底，已显式配置 Header 时尊重调用方的 TrustInbound）。
func (t *TraceIDConfig) applyDefaults() {
	if t.Header == "" {
		t.Header = TraceIDHeader
		t.TrustInbound = true
	}
}
