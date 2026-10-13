package constant

// ClientIDHeader 客户端标识请求头名。
// 登录接口的 clientId 走 JSON 体（LoginBody.ClientID），但记登录日志时拿不到已解析的
// body（失败分支在 body 解析前后都可能触发），故从请求头取。
//
// 留在共享层是因为 middleware/operlog 也要读它；其余通用常量已迁至各业务模块。
const ClientIDHeader = "clientid"
