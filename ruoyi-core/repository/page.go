package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/HandsomeNPC/ruoyi-common-go/ruoyi-core/model/page"
)

// ErrPageMissingModel 调用方未指定查询目标表。
var ErrPageMissingModel = errors.New("repository: 分页查询须由调用方指定 Model 或 Table")

// SelectPage 执行分页查询：先 COUNT 总数，再按分页与排序取当页数据。
//
// db 须由调用方拼好 Model/Table、WHERE、JOIN；本函数只补 COUNT、ORDER BY、LIMIT/OFFSET。
// 不从 dest 反推表名：dest 常是 VO 而非实体，反推会打到不存在的表、且丢掉逻辑删除
// 条件，故缺 Model/Table 时直接报错而不静默兜底。
func SelectPage[T any](db *gorm.DB, q page.PageQuery, dest *[]T) (page.PageResult[T], error) {
	if db.Statement.Model == nil && db.Statement.Table == "" {
		return page.EmptyPage[T](), ErrPageMissingModel
	}

	order, err := q.OrderBy()
	if err != nil {
		return page.EmptyPage[T](), err
	}

	// 新建 session，使 Count 与 Find 各自克隆 statement，避免 SELECT count(*) 污染 Find。
	session := db.Session(&gorm.Session{})

	var total int64
	if err := session.Count(&total).Error; err != nil {
		return page.EmptyPage[T](), fmt.Errorf("repository: 分页统计总数失败: %w", err)
	}
	if total == 0 {
		// 清空调用方的切片：复用同一个 dest 翻页时，不清会留下上一页的数据。
		*dest = (*dest)[:0]
		return page.EmptyPage[T](), nil
	}

	tx := session.Scopes(q.Paginate())
	if len(order.Columns) > 0 {
		tx = tx.Order(order)
	}
	if err := tx.Find(dest).Error; err != nil {
		return page.EmptyPage[T](), fmt.Errorf("repository: 分页查询数据失败: %w", err)
	}

	return page.Page(*dest, total), nil
}

// SelectPageCtx 语义同 SelectPage，额外绑定 ctx。
func SelectPageCtx[T any](ctx context.Context, db *gorm.DB, q page.PageQuery, dest *[]T) (page.PageResult[T], error) {
	return SelectPage(db.WithContext(ctx), q, dest)
}
