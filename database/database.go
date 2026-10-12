// Package database 多数据源 GORM 初始化与包级默认实例访问。
//
// 形态对照 Java RuoYi 的 spring.datasource.dynamic：primary 指定默认数据源，
// DB() 返回它；strict=true 时 DBByName 命中不到名字直接报错，否则回退 primary。
// 驱动按 type 分派，mysql/postgres/sqlserver 默认编译进包，oracle 经 -tags oracle 启用。
package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/repository"
)

// pingTimeout 启动时探活数据库的超时时间。
const pingTimeout = 5 * time.Second

// sourceEntry 一个命名数据源的运行时状态。
// lazy 源在 Init 时只登记、不建连，首次 DBByName 经 mu 守卫建连并置 opened。
type sourceEntry struct {
	cfg    SourceConfig
	mu     sync.Mutex // 仅 lazy 源用：守卫首次建连
	db     *gorm.DB
	opened bool
}

// manager Init 时刻冻结的只读快照。读热路径 DB() 只 holder.Load() 一次拿到
// primaryDB，无锁。lazy 源首次建连就地改 entry.db（经 entry.mu 保护），快照不重建。
type manager struct {
	primary   string
	strict    bool
	entries   map[string]*sourceEntry
	primaryDB *gorm.DB // entries[primary].db 的快照，DB() 直接返回
}

var holder atomic.Pointer[manager]

// Init 按配置建连并设置为包级默认实例，须在配置加载之后、开始接请求之前调用。
// 数据库是硬依赖，配置非法或主库建连失败直接 panic，fail-fast。
func Init(cfg DatasourceConfig) {
	if err := cfg.Validate(); err != nil {
		panic(fmt.Errorf("database: 配置非法: %w", err))
	}
	m := &manager{
		primary: cfg.Primary,
		strict:  cfg.Strict,
		entries: make(map[string]*sourceEntry, len(cfg.Datasource)),
	}

	// 先建非 lazy 源；lazy 源只登记，留待 DBByName 首次建连。
	// 任意源失败：回收已开连接后 panic。
	for name, sc := range cfg.Datasource {
		entry := &sourceEntry{cfg: sc}
		if sc.Lazy && name != cfg.Primary {
			m.entries[name] = entry
			continue
		}
		db, err := openSource(name, sc)
		if err != nil {
			m.cleanup()
			panic(fmt.Errorf("database: 初始化数据源 %q 失败: %w", name, err))
		}
		entry.db = db
		entry.opened = true
		m.entries[name] = entry
	}
	m.primaryDB = m.entries[cfg.Primary].db

	if old := holder.Swap(m); old != nil {
		// 误重复 Init：关掉旧快照里已开的连接，避免泄漏。
		old.cleanup()
	}
	log.Printf("[database] 已连接 %d 个数据源，主库=%s", len(m.entries), m.primary)
}

// DB 返回主库实例，未调用 Init 会 panic。读热路径，无锁。
func DB() *gorm.DB {
	m := holder.Load()
	if m == nil {
		panic("database: 尚未初始化，请先调用 database.Init")
	}
	return m.primaryDB
}

// DBByName 按名取数据源。
// 命中已开→返回；命中 lazy 未开→经 mu 守卫建连后返回（失败可下次重试）；
// 未命中→strict 返回 error，否则回退主库（与 Java dynamic-datasource strict=false 一致）。
func DBByName(name string) (*gorm.DB, error) {
	m := holder.Load()
	if m == nil {
		return nil, fmt.Errorf("database: 尚未初始化，请先调用 database.Init")
	}
	entry, ok := m.entries[name]
	if !ok {
		if m.strict {
			return nil, fmt.Errorf("database: 数据源 %q 不存在(strict=true)", name)
		}
		return m.primaryDB, nil
	}
	// 快路径：已开直接返回（无锁读，已开状态在 Init 后只读不再变；lazy 开后的
	// 写入经 entry.mu 释放建立 happens-before，读到的 db 必然可见）。
	if entry.opened {
		return entry.db, nil
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.opened {
		return entry.db, nil
	}
	db, err := openSource(name, entry.cfg)
	if err != nil {
		return nil, fmt.Errorf("database: 懒加载数据源 %q 失败: %w", name, err)
	}
	entry.db = db
	entry.opened = true
	return db, nil
}

// Close 关闭全部已开连接池并清空包级默认实例，供进程退出时 defer 调用。
func Close() {
	m := holder.Swap(nil)
	if m == nil {
		return
	}
	m.cleanup()
}

func (m *manager) cleanup() {
	for name, e := range m.entries {
		if e.opened {
			if err := closeDB(e.db); err != nil {
				log.Printf("[database] 关闭数据源 %q 失败: %v", name, err)
			}
		}
	}
}

// openSource 按配置建连一个数据源：分派驱动、配连接池、探活、注册审计回调。
func openSource(name string, cfg SourceConfig) (*gorm.DB, error) {
	opener, ok := lookupOpener(cfg.Type)
	if !ok {
		return nil, fmt.Errorf("database: 驱动 %q 未注册（默认构建支持 mysql/postgres/sqlserver；oracle 需 -tags oracle）", cfg.Type)
	}
	db, err := gorm.Open(opener(cfg.URL), &gorm.Config{
		Logger: newLogger(cfg),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		return nil, fmt.Errorf("database: 连接 %s(%s) 失败: %w", name, cfg.Type, err)
	}
	if err := setupPool(db, cfg); err != nil {
		_ = closeDB(db)
		return nil, err
	}
	if err := ping(db); err != nil {
		_ = closeDB(db)
		return nil, err
	}
	// 每个实例都注册审计回调：任何 main 都无法漏掉审计字段填充。
	if err := repository.RegisterAuditCallbacks(db); err != nil {
		_ = closeDB(db)
		return nil, fmt.Errorf("database: 注册审计字段回调失败: %w", err)
	}
	return db, nil
}

// setupPool 应用连接池参数。
func setupPool(db *gorm.DB, cfg SourceConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: 获取底层连接池失败: %w", err)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.MaxLifetime())
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.MaxIdleTime())
	}
	return nil
}

// ping 启动探活，确认数据库真实可用。
func ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: 获取底层连接池失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database: 数据库探活失败: %w", err)
	}
	return nil
}

// closeDB 关闭单个连接池。
func closeDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: 获取底层连接池失败: %w", err)
	}
	return sqlDB.Close()
}
