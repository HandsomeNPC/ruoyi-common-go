package repository

import (
	"context"
	"reflect"
	"time"

	"gorm.io/gorm"
)

// AuditDefaultUserID 取不到登录用户时写入的兜底值。
const AuditDefaultUserID int64 = -1

// 审计列名。回调按列名而非 Go 类型识别字段，因此 pkg/repository 无需 import
// 实体所在的 internal 包（那会成环），且不含这些列的表天然不受影响。
const (
	colCreateDept = "create_dept"
	colCreateBy   = "create_by"
	colCreateTime = "create_time"
	colUpdateBy   = "update_by"
	colUpdateTime = "update_time"
)

// AuditUser 审计字段的来源身份。
type AuditUser struct {
	UserID int64
	DeptID int64
}

type auditUserCtxKey struct{}

// WithAuditUser 把操作者身份放进 ctx，供审计回调取用。
// 由 HTTP 层的中间件调用——回调只拿得到 *gorm.DB，登录态只能靠 ctx 接力。
func WithAuditUser(ctx context.Context, u AuditUser) context.Context {
	return context.WithValue(ctx, auditUserCtxKey{}, u)
}

// AuditUserFrom 取出 ctx 中的操作者身份。
func AuditUserFrom(ctx context.Context) (AuditUser, bool) {
	if ctx == nil {
		return AuditUser{}, false
	}
	u, ok := ctx.Value(auditUserCtxKey{}).(AuditUser)
	return u, ok
}

// RegisterAuditCallbacks 注册审计字段自动填充回调。
//
// 两者都挂在 Before("gorm:create"/"gorm:update")，即用户钩子之后、gorm 读字段拼
// SQL 之前，这是唯一既能看到调用方预设值、又来得及改写的位置。
func RegisterAuditCallbacks(db *gorm.DB) error {
	if err := db.Callback().Create().Before("gorm:create").
		Register("ruoyi:audit_insert_fill", auditInsertFill); err != nil {
		return err
	}
	return db.Callback().Update().Before("gorm:update").
		Register("ruoyi:audit_update_fill", auditUpdateFill)
}

// resolveAuditUser 解析操作者，缺失的字段回落 -1。
//
// 逐字段回落而非整体回落：登录用户存在但未分配部门时 DeptID 为 0，而 0 号部门
// 并不存在，写进 create_dept 会造出一条指向空部门的脏数据。对照 Java
// InjectionMetaObjectHandler，createDept 也是独立兜底的。
func resolveAuditUser(ctx context.Context) AuditUser {
	u, _ := AuditUserFrom(ctx)
	if u.UserID == 0 {
		u.UserID = AuditDefaultUserID
	}
	if u.DeptID == 0 {
		u.DeptID = AuditDefaultUserID
	}
	return u
}

// auditInsertFill 插入前填充创建/更新审计字段。
//
// 逐行反射赋值而非走 stmt.SetColumn：后者在回调期会拿同一个值遍历整个 slice，
// 批量插入时会把各行预设的 create_time 冲掉。
func auditInsertFill(db *gorm.DB) {
	stmt := db.Statement
	if db.Error != nil || stmt == nil || stmt.Schema == nil || stmt.SkipHooks {
		return
	}

	now := time.Now()
	user := resolveAuditUser(stmt.Context)

	eachRow(stmt.ReflectValue, func(row reflect.Value) {
		// create_time 已有值时以它为基准，让 update_time 与之一致。
		current := now
		if f := stmt.Schema.FieldsByDBName[colCreateTime]; f != nil {
			if v, zero := f.ValueOf(stmt.Context, row); !zero {
				if t, ok := asTime(v); ok {
					current = t
				}
			}
			_ = f.Set(stmt.Context, row, current)
		}
		if f := stmt.Schema.FieldsByDBName[colUpdateTime]; f != nil {
			_ = f.Set(stmt.Context, row, current)
		}

		// 已指定创建人则三个人员字段整体不动。
		createBy := stmt.Schema.FieldsByDBName[colCreateBy]
		if createBy == nil {
			return
		}
		if _, zero := createBy.ValueOf(stmt.Context, row); !zero {
			return
		}
		_ = createBy.Set(stmt.Context, row, user.UserID)
		if f := stmt.Schema.FieldsByDBName[colUpdateBy]; f != nil {
			_ = f.Set(stmt.Context, row, user.UserID)
		}
		if f := stmt.Schema.FieldsByDBName[colCreateDept]; f != nil {
			if _, zero := f.ValueOf(stmt.Context, row); zero {
				_ = f.Set(stmt.Context, row, user.DeptID)
			}
		}
	})
}

// auditUpdateFill 更新前无条件刷新 update_by/update_time。
//
// 这里必须用 SetColumn：Updates(map) / Update(col, val) 的 Dest 是 map，逐行反射覆盖不到。
func auditUpdateFill(db *gorm.DB) {
	stmt := db.Statement
	if db.Error != nil || stmt == nil || stmt.Schema == nil || stmt.SkipHooks {
		return
	}

	// SetColumn 对 map 型 Dest 是原地写入，会把审计列塞进调用方自己的 map。
	// 调用方若复用该 map（service 普遍先 build 再按条件补键），上一轮的 update_time
	// 会被当成业务列带进下一轮，故先换成副本再填。
	detachMapDest(stmt)

	if stmt.Schema.FieldsByDBName[colUpdateTime] != nil {
		stmt.SetColumn(colUpdateTime, time.Now(), true)
	}
	if stmt.Schema.FieldsByDBName[colUpdateBy] != nil {
		stmt.SetColumn(colUpdateBy, resolveAuditUser(stmt.Context).UserID, true)
	}
}

// detachMapDest 把 map 型 Dest 替换成浅拷贝，隔离后续 SetColumn 对调用方的副作用。
// 非 map 的 Dest（struct/slice）不受影响：那条路径 SetColumn 写的是 ReflectValue，
// 本就指向调用方的实体，刷新 update_by 到实体上是预期行为。
func detachMapDest(stmt *gorm.Statement) {
	switch dest := stmt.Dest.(type) {
	case map[string]any:
		cp := make(map[string]any, len(dest)+2)
		for k, v := range dest {
			cp[k] = v
		}
		stmt.Dest = cp
	case []map[string]any:
		cp := make([]map[string]any, len(dest))
		for i, m := range dest {
			row := make(map[string]any, len(m)+2)
			for k, v := range m {
				row[k] = v
			}
			cp[i] = row
		}
		stmt.Dest = cp
	}
}

// eachRow 对 Struct 目标回调一次，对 Slice/Array 目标逐元素回调。
func eachRow(rv reflect.Value, fn func(reflect.Value)) {
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			row := reflect.Indirect(rv.Index(i))
			if row.Kind() == reflect.Struct {
				fn(row)
			}
		}
	case reflect.Struct:
		if rv.CanAddr() {
			fn(rv)
		}
	}
}

// asTime 从字段值取出 time.Time，兼容 *time.Time 字段。
func asTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case *time.Time:
		if t != nil {
			return *t, true
		}
	}
	return time.Time{}, false
}
