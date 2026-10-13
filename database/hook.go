package database

import (
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// OpenHook 在一个数据源建连、配好连接池并探活之后执行，用于给实例挂 GORM
// 回调/插件（审计字段填充、逻辑删除等）。返回 error 视为该数据源初始化失败。
type OpenHook func(db *gorm.DB) error

// 本包只做数据库初始化，不认识任何业务语义；需要往每个实例挂回调的上层包
// （如 ruoyi-core/repository 的审计字段填充）在 init() 里登记到这里，依赖方向
// 保持业务→基础设施的单向。
var (
	hookMu sync.RWMutex
	hooks  []namedHook
)

type namedHook struct {
	name string
	fn   OpenHook
}

// RegisterOpenHook 登记建连后钩子，按注册顺序执行。
// 须在 Init 之前调用（通常在包的 init() 里）：Init 之后再注册只对此后懒加载
// 建连的数据源生效，已建好的实例不会补挂。
// name 仅用于报错定位；重复注册同名钩子 panic，避免重复挂同一组回调。
func RegisterOpenHook(name string, fn OpenHook) {
	if name == "" || fn == nil {
		panic("database: RegisterOpenHook 的 name 与 fn 均不能为空")
	}
	hookMu.Lock()
	defer hookMu.Unlock()
	for _, h := range hooks {
		if h.name == name {
			panic(fmt.Errorf("database: 建连钩子 %q 重复注册", name))
		}
	}
	hooks = append(hooks, namedHook{name: name, fn: fn})
}

// applyOpenHooks 依次执行已登记的钩子，任一失败即中断并返回。
func applyOpenHooks(db *gorm.DB) error {
	hookMu.RLock()
	snapshot := hooks
	hookMu.RUnlock()

	for _, h := range snapshot {
		if err := h.fn(db); err != nil {
			return fmt.Errorf("database: 执行建连钩子 %q 失败: %w", h.name, err)
		}
	}
	return nil
}
