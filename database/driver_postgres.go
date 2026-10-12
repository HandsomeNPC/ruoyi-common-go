package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	registerDriver(DriverPostgres, func(dsn string) gorm.Dialector { return postgres.Open(dsn) })
}
