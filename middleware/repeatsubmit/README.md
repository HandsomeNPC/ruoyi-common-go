# repeatsubmit

防重复提交注解（装饰器），防重键存 Redis。对应 Java RuoYi 的 `@RepeatSubmit` 注解。

对应原 `ruoyi-core/repeatsubmit`，抽出为独立包，与 `ratelimiter` 同属注解型中间件。

## 初始化

防重键走 Redis，`Init` 须在 `redis.Init` 之后、路由注册之前调用，并传入 Redis 客户端与 token 请求头名（取 satoken 配置）：

```go
import (
    "github.com/HandsomeNPC/ruoyi-common-go/redis"
    "github.com/HandsomeNPC/ruoyi-common-go/middleware/repeatsubmit"
)

redis.Init(cfg.Redis)
repeatsubmit.Init(redis.Client(), cfg.SAToken.TokenName)
```

未 `Init` 就注册防重路由会在路由装配那一刻 panic（而非拖到首个请求才崩）；`Init` 收到 nil redis 客户端同样 panic，不在此兜底包级
`redis.Client()`，nil 越早暴露越好定位。

## 判定逻辑

参考美团 GTIS 防重：

1. **进 handler 前** 用 `SETNX` 抢防重键，抢不到即判为重复提交并拦截。
2. **handler 成功**（响应 `code=200`）则保留键，interval 内的重复请求都被挡住。
3. **handler 失败或 panic** 则删键，允许立刻重试。

## 用法

签名：`RepeatSubmit(interval, message)`，零值取默认（间隔 5s、提示「不允许重复提交，请稍候再试」）；
`interval < 1 秒` 在注册期 panic。

```go
// 新增用户：默认 5s 防重
g.POST("/user", repeatsubmit.RepeatSubmit(0, ""), handler.UserApiApp.Add)

// 自定义 10s 间隔
g.POST("/user", repeatsubmit.RepeatSubmit(10*time.Second, ""), handler.UserApiApp.Add)
```

> 须挂在 `apiencrypt` 之后：指纹取的是请求体，须是解密后的明文，否则密文每次随机 AES 密钥、
> 同样入参会算出不同指纹，防重直接失效。

## 指纹

`global:repeat_submit:<请求路径><sha256(token+":"+入参)>`，入参 = 请求体 + query 串（无 AOP 取不到方法入参，
故用报文体代替）。请求体优先取 `gin.BodyBytesKey` 里 `RepeatableBody` 缓存的副本；未缓存时读后塞回 Body， handler 仍能完整读到；超过
1MB 的部分不参与指纹。

## 行为约定

- **重复提交**触发 `c.Error(errs.New(0, message, ""))` 并 `Abort`，由全局异常中间件渲染文案。
- **Redis 异常时放行**：可用性优先，不因防重器故障拖垮接口；异常已记日志。
- **响应成功才保留键**：缓冲响应体判定业务 `code`，非 R 结构（文件流、空响应等）按成功处理（保留键）， 对照 Java `instanceof R`
  不匹配即放行。
- **删键用 `WithoutCancel`**：客户端断连会取消请求 context，但键仍要删，否则重连后白等一个 interval。
