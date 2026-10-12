# repeatablebody

可重复读请求体中间件。把配置的 content-type 的请求体读入内存并塞回
`gin.BodyBytesKey`，让后续中间件（XSS 清洗、访问日志、加密解密、防重提交指纹） 与 handler 都能重复读 body。超出 `MaxBodySize`
的请求直接拒绝。

对应原 `ruoyi-core/middleware/body.go`，抽出为独立包。

## 用法

注册在 TraceID 之后、AccessLog / XSS 之前：

```go
import (
"github.com/HandsomeNPC/ruoyi-common-go/middleware"
"github.com/HandsomeNPC/ruoyi-common-go/middleware/repeatablebody"
)

r.Use(
middleware.Recover(),
middleware.CORSWithConfig(cfg.CORS),
middleware.TraceIDWithConfig(cfg.TraceID),
repeatablebody.RepeatableBodyWithConfig(cfg.RepeatableBody),
middleware.AccessLogWithConfig(cfg.AccessLog),
middleware.XSSWithConfig(cfg.XSS),
)
```

下游取已缓存的请求体（直接走 gin 上下文，无需本包）：

```go
v, _ := c.Get(gin.BodyBytesKey)
body, _ := v.([]byte)
```

## 配置

| 字段           | 默认值                 | 说明                                       |
|----------------|------------------------|--------------------------------------------|
| `contentTypes` | `["application/json"]` | 需缓存的 content-type 前缀，大小写不敏感。 |
| `maxBodySize`  | 10MB                   | 超出则拒绝请求。                           |

零值字段回落默认。`SetDefaults` / `Validate` 与其它中间件配置一致，供各模块
`config` 统一铺默认值与校验。

## 行为约定

- **超限**：直接 `response.Fail("请求体超出大小限制")` 写回（HTTP 200、code 500）， 不走 `errs` + `Recover` 兜底，避免引入
  errs 依赖与隐式时序。
- **读取失败**：登记原始 `error` 由全局 `Recover` 兜底渲染（带错误编号）， 日志带 traceId。
- **多读 1 字节**判定上限：能区分「刚好到上限」与「已经超了」，避免边界误判。
- **仅缓存声明的 content-type**：multipart/二进制等不进内存缓存。
