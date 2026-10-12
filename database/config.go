package database

import (
	"fmt"
	"time"
)

// 数据源驱动类型。oracle 默认构建不编译进包（依赖 CGO + Oracle Instant
// Client），需用 -tags oracle 构建；mysql/postgres/sqlserver 均为纯 Go 驱动。
const (
	DriverMySQL     = "mysql"
	DriverPostgres  = "postgres"
	DriverOracle    = "oracle"
	DriverSQLServer = "sqlserver"
)

// SQL 日志级别取值。
const (
	LogLevelSilent = "silent"
	LogLevelError  = "error"
	LogLevelWarn   = "warn"
	LogLevelInfo   = "info"
)

// SourceConfig 单个命名数据源配置，对应 yaml 的 datasource.datasource.<name> 段。
// URL 为完整的 GORM DSN，账号密码含在其中，驱动 Open 只吃这一串。
type SourceConfig struct {
	Type            string `mapstructure:"type"` // mysql|postgres|sqlserver|oracle
	URL             string `mapstructure:"url"`  // GORM DSN
	Lazy            bool   `mapstructure:"lazy"` // 懒加载，首次 DBByName 才建连
	MaxIdleConns    int    `mapstructure:"maxIdleConns"`
	MaxOpenConns    int    `mapstructure:"maxOpenConns"`
	ConnMaxLifetime int    `mapstructure:"connMaxLifetime"` // 秒
	ConnMaxIdleTime int    `mapstructure:"connMaxIdleTime"` // 秒
	LogLevel        string `mapstructure:"logLevel"`        // silent/error/warn/info，缺省 warn
	SlowThresholdMs int    `mapstructure:"slowThresholdMs"` // 慢 SQL 阈值(毫秒)，缺省 200
}

// MaxLifetime 返回连接最大存活时长。
func (s SourceConfig) MaxLifetime() time.Duration {
	return time.Duration(s.ConnMaxLifetime) * time.Second
}

// MaxIdleTime 返回空闲连接最大存活时长。
func (s SourceConfig) MaxIdleTime() time.Duration {
	return time.Duration(s.ConnMaxIdleTime) * time.Second
}

// SlowThreshold 返回慢 SQL 阈值，未配置时默认 200ms。
func (s SourceConfig) SlowThreshold() time.Duration {
	if s.SlowThresholdMs <= 0 {
		return 200 * time.Millisecond
	}
	return time.Duration(s.SlowThresholdMs) * time.Millisecond
}

// Level 返回规范化后的 SQL 日志级别，未配置时默认 warn。
func (s SourceConfig) Level() string {
	if s.LogLevel == "" {
		return LogLevelWarn
	}
	return s.LogLevel
}

// supportedDriver 是否为已注册驱动。oracle 在默认构建下未注册（见 driver_oracle.go）。
func supportedDriver(t string) bool {
	switch t {
	case DriverMySQL, DriverPostgres, DriverSQLServer, DriverOracle:
		return true
	}
	return false
}

// Validate 校验单个数据源配置。
func (s SourceConfig) Validate() error {
	if !supportedDriver(s.Type) {
		return fmt.Errorf("datasource.type 必须为 %s/%s/%s/%s",
			DriverMySQL, DriverPostgres, DriverSQLServer, DriverOracle)
	}
	if s.URL == "" {
		return fmt.Errorf("datasource.url 未配置")
	}
	if s.MaxOpenConns > 0 && s.MaxIdleConns > s.MaxOpenConns {
		return fmt.Errorf("datasource.maxIdleConns 不能大于 maxOpenConns")
	}
	switch s.LogLevel {
	case "", LogLevelSilent, LogLevelError, LogLevelWarn, LogLevelInfo:
	default:
		return fmt.Errorf("datasource.logLevel 必须为 silent/error/warn/info 之一")
	}
	return nil
}

// DatasourceConfig 多数据源配置，对应 yaml 的 datasource 段（primary/strict/datasource）。
// primary 指定默认数据源名，DB() 即返回它；strict 为 true 时 DBByName 命中不到名字直接报错，
// 为 false 时回退 primary（与 Java dynamic-datasource 语义一致）。
type DatasourceConfig struct {
	Primary    string                  `mapstructure:"primary"`
	Strict     bool                    `mapstructure:"strict"`
	Datasource map[string]SourceConfig `mapstructure:"datasource"`
}

// Validate 校验多数据源配置：primary 非空且存在、至少一个源、每个源自身合法。
func (d DatasourceConfig) Validate() error {
	if d.Primary == "" {
		return fmt.Errorf("datasource.primary 未配置")
	}
	if len(d.Datasource) == 0 {
		return fmt.Errorf("datasource.datasource 至少配置一个数据源")
	}
	if _, ok := d.Datasource[d.Primary]; !ok {
		return fmt.Errorf("datasource.primary=%q 在 datasource 中不存在", d.Primary)
	}
	for name, src := range d.Datasource {
		if err := src.Validate(); err != nil {
			return fmt.Errorf("datasource[%s]: %w", name, err)
		}
	}
	return nil
}
