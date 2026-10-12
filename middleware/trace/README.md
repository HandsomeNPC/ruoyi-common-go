# trace

链路 id 中间件。取或生成 traceId 写进 `gin.Context` 与 `context.Context`，并回写响应头， 供下游日志与调用方对齐。

对应原 `ruoyi-core/middleware/trace.go`，抽出为独立包。

## 用法

注册在 `CORS` 之后、`RepeatableBody` 之前：

```go
import (
	"github.com/HandsomeNPC/ruoyi-common-go/middleware/cors"
	"github.com/HandsomeNPC/ruoyi-common-go/middleware/trace"
)

r.Use(
	cors.CORSWithConfig(cfg.CORS),
	trace.TraceIDWithConfig(cfg.TraceID),
)
```

`TraceIDFrom(ctx)` 从 `context.Context` 取 id（不依赖 gin.Context）， 供 service/repository 等非 gin 代码打日志用。

## 配置

| 字段           | 默认值         | 说明                                          |
|----------------|----------------|-----------------------------------------------|
| `header`       | `X-Request-Id` | 读写链路 id 的头名。                          |
| `trustInbound` | `true`         | 是否沿用入站请求头里已有的 id（白名单校验）。 |

零值字段回落默认。`SetDefaults` / `Validate` 与其它中间件配置一致， 供各模块 `config` 统一铺默认值与校验。

## 行为约定

- **必须注册在 `CORS` 之后**：CORS 先写自己的头，本中间件再追加 traceId 头，避免被覆盖。
- **入站 id 走白名单而非过滤**：不合规矩（超长或含非法字符）直接丢弃重新生成， 避免把两个不同的入站 id 过滤后折叠成同一个，导致跨请求串链。
- **无条件回写响应头**：调用方能确认服务端最终认下了哪个 id，且必须在 `c.Next()` 之前写。
- **本包不依赖 `ruoyi-core/middleware`**：traceId 取值与日志前缀就地复刻， 与 `accesslog`/`xss`/`repeatablebody`
  一致，换取中间件包的完全自包含。
