# ruoyi-common-go

从 [RuoYi-Go-Vue-Plus](https://github.com/HandsomeNPC/RuoYi-Go-Vue-Plus) 中抽出的通用工具包，供其他 Go 项目按需复用。

## 子包

| 包                  | 用途                                                                                                  |
|---------------------|-------------------------------------------------------------------------------------------------------|
| bcrypt              | 密码哈希（hutool BCrypt 对齐：Hashpw 自动产盐 + Verify/Checkpw 校验，跨语言兼容 Java 侧哈希）         |
| ip                  | 客户端 IP 识别（代理头优先级 + 离线归属地解析）                                                       |
| jsonx               | 雪花 ID 安全的 JSON 编解码（超 JS 安全整数范围转字符串）+ 接管 gin 全局 codec                         |
| snowflake           | 雪花算法 ID 生成器（Twitter 64 位布局，时钟回拨等待追平，无锁包级实例）                               |
| strutil             | 字符串工具（首字母大写、http(s) 链接判定、逗号分隔主键串解析）                                        |
| treeutil            | 树形结构构建（单根/多根挂接 + 就地拼装，雪花 ID 安全的 JSON 序列化）                                  |
| useragent           | User-Agent 解析（浏览器名 + 操作系统名）                                                              |
| sms                 | 短信发送（Sender 接口 + 内置阿里云实现，走官方 dysmsapi-20170525 SDK；换/加厂商再写一个 Sender 即可） |
| ruoyi-core/constant | 业务常量（基础常量、全局 Redis key 前缀、缓存组名/TTL、推送类型、常用正则与密码校验、系统常量）       |


## 安装

```bash
go get github.com/HandsomeNPC/ruoyi-common-go
```

## License

与主仓 RuoYi-Go-Vue-Plus 保持一致。
