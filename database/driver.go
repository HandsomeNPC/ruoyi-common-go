package database

import "gorm.io/gorm"

// DialectorOpener 按 DSN 构造 GORM Dialector。每个驱动在 init 时注册自己。
// 用注册表而非裸 switch，是为了让 oracle（依赖 CGO + Oracle Instant Client）
// 能经构建标签隔离：默认构建只编译进 mysql/postgres/sqlserver 三个纯 Go 驱动，
// oracle 由 driver_oracle.go（//go:build oracle）单独注册。
type DialectorOpener func(dsn string) gorm.Dialector

var openers = map[string]DialectorOpener{}

func registerDriver(typ string, op DialectorOpener) {
	openers[typ] = op
}

// lookupOpener 取驱动 opener；未注册时 ok=false，调用方据此给出"未编译进本构建"的提示。
func lookupOpener(typ string) (DialectorOpener, bool) {
	op, ok := openers[typ]
	return op, ok
}
