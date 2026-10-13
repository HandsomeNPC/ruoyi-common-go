package repository

import "strings"

// likeEscaper 转义 LIKE 模式里的元字符，使入参按字面量匹配。
// 不转义的话搜 "%" 会命中全表、"_" 会变成任意单字符通配。
// NewReplacer 单遍扫描、不回溯已替换的内容，故三条规则互不干扰，顺序无关。
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// EscapeLike 转义 LIKE 模式里的元字符，返回可安全拼进 LIKE 模式的字面量。
//
// 转义符用反斜杠，依赖 MySQL 的默认行为，调用方无需另写 ESCAPE 子句。
// 若将来接 PostgreSQL（standard_conforming_strings）或 MySQL 开了
// NO_BACKSLASH_ESCAPES，反斜杠不再是 LIKE 转义符，须改为显式
// `LIKE ? ESCAPE '\'` 并在此同步调整。
func EscapeLike(s string) string {
	return likeEscaper.Replace(s)
}
