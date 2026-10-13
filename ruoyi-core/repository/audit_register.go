package repository

import (
	"github.com/HandsomeNPC/ruoyi-common-go/database"
)

// 审计回调的注册在本包这侧完成：database 只负责建连，不认识审计语义。
// 依赖方向是 repository → database，链接了本包的进程自动对每个数据源生效，
// 各 main 无需显式调用，也无从漏掉。
func init() {
	database.RegisterOpenHook("ruoyi:audit", RegisterAuditCallbacks)
}
