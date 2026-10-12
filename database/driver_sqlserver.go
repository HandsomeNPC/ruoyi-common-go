package database

import (
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

func init() {
	registerDriver(DriverSQLServer, func(dsn string) gorm.Dialector { return sqlserver.Open(dsn) })
}
