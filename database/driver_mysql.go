package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func init() {
	registerDriver(DriverMySQL, func(dsn string) gorm.Dialector { return mysql.Open(dsn) })
}
