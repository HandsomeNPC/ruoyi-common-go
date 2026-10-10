# ruoyi-common-go

从 [RuoYi-Go-Vue-Plus](https://github.com/HandsomeNPC/RuoYi-Go-Vue-Plus) 中抽出的通用工具包，供其他 Go 项目按需复用。

## 子包

| 包        | 用途                                                                          |
|-----------|-------------------------------------------------------------------------------|
| ip        | 客户端 IP 识别（代理头优先级 + 离线归属地解析）                               |
| jsonx     | 雪花 ID 安全的 JSON 编解码（超 JS 安全整数范围转字符串）+ 接管 gin 全局 codec |
| strutil   | 字符串工具（首字母大写、http(s) 链接判定、逗号分隔主键串解析）                |
| useragent | User-Agent 解析（浏览器名 + 操作系统名）                                      |


## 安装

```bash
go get github.com/HandsomeNPC/ruoyi-common-go
```

## License

与主仓 RuoYi-Go-Vue-Plus 保持一致。
