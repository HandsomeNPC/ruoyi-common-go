# ratelimiter

接口限流（滑动窗口），计数存 Redis。对应 Java RuoYi 的 `@RateLimiter` 注解。

## 初始化

限流计数走 Redis，`Init` 须在 `redis.Init` 之后、路由注册之前调用，并传入 Redis 客户端：

```go
import (
    "github.com/HandsomeNPC/ruoyi-common-go/redis"
    "github.com/HandsomeNPC/ruoyi-common-go/middleware/ratelimiter"
)

redis.Init(cfg.Redis)
ratelimiter.Init(redis.Client())
```

未 `Init` 就注册限流路由会在路由装配那一刻 panic（而非拖到首个请求才崩）。

## 三种限流维度

| 类型               | 维度             | 适用                           |
|--------------------|------------------|--------------------------------|
| `LimitTypeGlobal`  | 全局共用一个额度 | 默认值；接口总并发上限         |
| `LimitTypeIP`      | 按请求方 IP      | 验证码接口防同一 IP 刷         |
| `LimitTypeCluster` | 按本实例独立额度 | 多实例各自独立，不做跨实例合计 |

## 用法

签名：`RateLimiter(window, count, limitType, timeout, message)`，零值取默认 （窗口 60s、100 次、键存活
24h、提示"访问过于频繁，请稍候再试"）。

```go
// 验证码接口：同一 IP 60 秒 10 次
r.GET("/auth/code",
    ratelimiter.RateLimiter(time.Minute, 10, ratelimiter.LimitTypeIP, 0, ""),
    handler.Code)
```

带动态维度（替代 Java 的 SpEL `#phoneNumber`）：传入闭包从请求取维度值， 返回空串表示不带。固定 `LimitTypeGlobal`
，维度以闭包返回值为准，不再叠加 IP / 实例。

```go
// 发短信验证码：同一手机号 60 秒 1 次
r.GET("/resource/sms/code",
    ratelimiter.RateLimiterWithKeyFunc(time.Minute, 1,
        func(c *gin.Context) string { return c.Query("phoneNumber") }, 0, ""),
    handler.SmsCode)
```

## 行为约定

- **滑动窗口**：自写 Lua（ZSET）单次 `EVAL` 原子完成"清过期→计数→判超→写入"，避免读-判-写竞态。
- **超限**触发 `c.Error(errs.New(0, message, ""))` 并 `Abort`，由全局异常中间件渲染文案。
- **Redis 异常时放行**：可用性优先，不因限流器故障拖垮接口；异常已记日志。
- **配置即时生效**：每次按当前参数计算，改 window/count 不需重启或清缓存。
