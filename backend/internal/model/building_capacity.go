package model

import "time"

// BuildingCapacity 楼栋当日访客在场容量配置（按楼栋一行）。
// 在场人数不在此表累加，而由 VisitorPass(status=checked_in 且当日进入) 实时统计，
// 因此访客离开或被逾期兜底后容量自动恢复，无需回写计数器。
type BuildingCapacity struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Building   string    `gorm:"uniqueIndex;size:50" json:"building"`
	DailyLimit int       `json:"daily_limit"`
	UpdatedBy  uint      `json:"updated_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
