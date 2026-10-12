# accesslog

访问日志中间件。按 content-type 分流打印入参：JSON 走 `RepeatableBody` 缓存的 body， 表单/query 走 `Form`，再用 `defer`
补一条结束日志带状态码与耗时。

对应原 `ruoyi-core/middleware/logger.go`，抽出为独立包。

## 用法

注册在 `TraceID` / `RepeatableBody` 之后、`XSS` 之前：

```go
import (
"github.com/HandsomeNPC/ruoyi-common-go/middleware"
"github.com/HandsomeNPC/ruoyi-common-go/middleware/accesslog"
"github.com/HandsomeNPC/ruoyi-common-go/middleware/cors"
"github.com/HandsomeNPC/ruoyi-common-go/middleware/repeatablebody"
"github.com/HandsomeNPC/ruoyi-common-go/middleware/trace"
"github.com/HandsomeNPC/ruoyi-common-go/middleware/xss"
)

r.Use(
middleware.Recover(),
cors.CORSWithConfig(cfg.CORS),
trace.TraceIDWithConfig(cfg.TraceID),
repeatablebody.RepeatableBodyWithConfig(cfg.RepeatableBody),
accesslog.AccessLogWithConfig(cfg.AccessLog),
xss.XSSWithConfig(cfg.XSS),
)
```

## 配置

| 字段             | 默认值 | 说明                                  |
|------------------|--------|---------------------------------------|
| `maxParamLength` | 4000   | 参数日志最大字符数，超出截断。        |
| `skipPaths`      | 空     | 不打日志的路径，精确匹配 `URL.Path`。 |

零值字段回落默认。`SetDefaults` / `Validate` 与其它中间件配置一致，供各模块 `config`
统一铺默认值与校验。

## 行为约定

- **必须注册在 `RepeatableBody` 之后**：JSON 入参从 `gin.BodyBytesKey` 取缓存 body，否则读不到。
- **路径取 `URL.Path`**：不用 `Request.RequestURI`，后者带原始查询串会把 `?password=xxx` 原样落日志。
- **按 content-type 分流入参**，不看请求方法。
- **参数脱敏在本包自带一份**（`param.go`）：敏感字段剔除、雪花 id 精度、截断都内置， 不引外部脱敏包，换取中间件包的完全自包含。本包不依赖
  `ruoyi-core/middleware`， traceId 取值与日志前缀就地复刻。
