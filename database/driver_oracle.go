//go:build oracle

// driver_oracle 仅在 -tags oracle 构建时编译进包。oracle 驱动经 godror 依赖
// CGO 与 Oracle Instant Client，默认纯 Go 构建不含它；需要 oracle 数据源时
// 先 `go get github.com/cengsin/oracle`，再 `go build -tags oracle`。
package database

import (
	"github.com/cengsin/oracle"
	"gorm.io/gorm"
)

func init() {
	registerDriver(DriverOracle, func(dsn string) gorm.Dialector { return oracle.Open(dsn) })
}
