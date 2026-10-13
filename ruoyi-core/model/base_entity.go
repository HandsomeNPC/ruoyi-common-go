package model

import "time"

// BaseEntity 审计字段，嵌入各模块实体。
// 具体值由 service 层在落库前按登录态写入；repository 审计回调按列名
// （create_by/create_dept/create_time/update_by/update_time）识别，不认类型。
type BaseEntity struct {
	CreateDept int64      `gorm:"column:create_dept" json:"createDept"`
	CreateBy   int64      `gorm:"column:create_by" json:"createBy"`
	CreateTime *time.Time `gorm:"column:create_time" json:"createTime"`
	UpdateBy   int64      `gorm:"column:update_by" json:"updateBy"`
	UpdateTime *time.Time `gorm:"column:update_time" json:"updateTime"`
}
