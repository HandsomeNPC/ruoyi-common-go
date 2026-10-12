# xss

请求清洗中间件。把请求里的 HTML 标签剔掉（保留标签内的文字），覆盖查询串、表单 与 JSON body。JSON 走逐值清洗避免破坏结构，body
取自 `RepeatableBody` 缓存的
`gin.BodyBytesKey`。

对应原 `ruoyi-core/middleware/xss.go`，抽出为独立包。

## 用法

注册在 `RepeatableBody` / `AccessLog` 之后、所有读参数的 handler 之前：

```go
import (
	"github.com/HandsomeNPC/ruoyi-common-go/middleware"
	"github.com/HandsomeNPC/ruoyi-common-go/middleware/accesslog"
	"github.com/HandsomeNPC/ruoyi-common-go/middleware/repeatablebody"
	"github.com/HandsomeNPC/ruoyi-common-go/middleware/trace"
	"github.com/HandsomeNPC/ruoyi-common-go/middleware/xss"
)

r.Use(
	middleware.Recover(),
	trace.TraceIDWithConfig(cfg.TraceID),
	repeatablebody.RepeatableBodyWithConfig(cfg.RepeatableBody),
	accesslog.AccessLogWithConfig(cfg.AccessLog),
	xss.XSSWithConfig(cfg.XSS),
)
```

## 配置

| 字段          | 默认值                                      | 说明                                      |
|---------------|---------------------------------------------|-------------------------------------------|
| `excludeUrls` | `["/system/notice","/warm-flow/save-json"]` | 跳过清洗的路径，Ant 风格。                |
| `skipMethods` | `["GET","DELETE"]`                          | 跳过清洗的请求方法，为空回落 GET/DELETE。 |

零值字段回落默认。默认值由 `XSSWithConfig` 注册时就地铺底，调用方无需手动调 `SetDefaults` / `Validate`。

## 行为约定

- **必须注册在 `RepeatableBody` 之后**：JSON body 从 `gin.BodyBytesKey` 取缓存，否则读不到。
- **JSON 逐值清洗**：递归遍历 JSON 树只洗字符串值，不清洗整串，避免破坏 JSON 结构；`json.Decoder` 用 `UseNumber`，否则 19 位雪花
  id 会被 float64 抹尾数。
- **快速跳过**：查询串/JSON body 里没有裸 `<`（查询串再加 `%`）时直接跳过，不做解析。
- **GET/DELETE 默认跳过**：查询串与表单多半是无害读参，清洗收益低且易误伤。
- **排除路径取 `URL.Path`**：不用 `Request.RequestURI`，否则带查询串时匹配不上排除规则。
- **Ant 路径匹配自带一份**（`path.go`）：与 `ExcludeURLs` 配套，本包不依赖 `ruoyi-core/middleware`，traceId 取值与日志前缀就地复刻。
- **清洗后回写三处同步**：`Request.Body`、`gin.BodyBytesKey`、`ContentLength`，且去掉 `json.Encoder` 补的换行符让长度与 body
  严格对上。
