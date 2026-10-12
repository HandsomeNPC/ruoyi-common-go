# cors

跨域中间件。按 origin 通配白名单放行来源，预检 (OPTIONS)回显允许的方法/头/缓存 时长，不合规的跨域请求直接 403。必须注册在
Auth 之前。

对应原 `ruoyi-core/middleware/cors.go`，抽出为独立包。

## 用法

注册在路由最前、Auth 之前：

```go
import (
"github.com/HandsomeNPC/ruoyi-common-go/middleware"
"github.com/HandsomeNPC/ruoyi-common-go/middleware/cors"
)

r.Use(
middleware.Recover(),
cors.CORSWithConfig(cfg.CORS),
middleware.TraceIDWithConfig(cfg.TraceID),
)
```

## 配置

| 字段                    | 默认值                                                       | 说明                                                    |
|-------------------------|--------------------------------------------------------------|---------------------------------------------------------|
| `allowCredentials`      | `true`                                                       | 是否允许携带凭证(cookie / Authorization)。              |
| `allowedOriginPatterns` | `["*"]`                                                      | 允许的来源，支持 `*` 通配，如 `https://*.example.com`。 |
| `allowedMethods`        | `["*"]`                                                      | 允许的请求方法，`["*"]` 表示回显预检请求的方法。        |
| `allowedHeaders`        | `["*"]`                                                      | 允许的请求头，`["*"]` 表示回显预检请求的头。            |
| `exposedHeaders`        | `["X-Request-Id","Content-Disposition","download-filename"]` | 允许前端 JS 读取的响应头。                              |
| `maxAgeSeconds`         | `1800`                                                       | 预检结果缓存时长(秒)。                                  |

`SetDefaults` / `Validate` 与其它中间件配置一致，供各模块 `config` 统一铺默认值与校验。

## 行为约定

- **Vary 提前加**：在判断是否跨域之前就写 `Vary: Origin / Access-Control-Request-Method /
  Access-Control-Request-Headers`，非跨域请求也带，避免 CDN 把跨域响应错缓成同域响应。
- **预检 = OPTIONS + `Access-Control-Request-Method`**：缺一不可，单纯 OPTIONS 当普通请求放行。
- **方法/头白名单 `["*"]`**：方法回显预检请求的方法、头回显预检请求的头；非通配则校验请求项 是否在白名单内，不合规整体 403。
- **头名大小写不敏感**（RFC 9110）。
- **来源通配**：`*` 全放；带 `*` 的 pattern 按首段贴开头、尾段贴结尾、中段按序定位匹配； 无 `*` 的 pattern 大小写不敏感全等。
- **不合规直接 403**：写回 `Invalid CORS request`，不走统一响应渲染（跨域响应本就不该被业务层兜底）。
- **预检响应才写** `Allow-Methods` / `Allow-Headers` / `Max-Age` 三个头，普通跨域请求不写。
