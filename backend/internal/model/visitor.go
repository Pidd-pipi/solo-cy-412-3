package model

import (
	"gorm.io/gorm"
	"time"
)

// VisitorPass 访客通行凭证。同一访客（手机号）在同一楼栋的重叠时段只允许存在一张有效凭证。
// 所有时刻以 UTC 绝对时刻入库（见 BeforeSave），跨时区部署与 SQLite/MySQL 的时间范围比较都一致；
// 对外展示由接口层统一按社区时区渲染。
type VisitorPass struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	PassNo         string     `gorm:"uniqueIndex;size:32" json:"pass_no"`
	ResidentID     uint       `gorm:"index" json:"resident_id"`
	Resident       User       `gorm:"foreignKey:ResidentID" json:"resident"`
	VisitorName    string     `gorm:"size:50" json:"visitor_name"`
	VisitorPhone   string     `gorm:"index;size:30" json:"visitor_phone"`
	Building       string     `gorm:"index;size:50" json:"building"`
	Reason         string     `json:"reason"`
	StartTime      time.Time  `gorm:"index" json:"start_time"`
	EndTime        time.Time  `gorm:"index" json:"end_time"`
	Status         string     `gorm:"index;size:20" json:"status"`
	ReviewerID     *uint      `json:"reviewer_id"`
	Reviewer       *User      `gorm:"foreignKey:ReviewerID" json:"reviewer,omitempty"`
	ReviewRemark   string     `json:"review_remark"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	CheckInAt      *time.Time `json:"check_in_at"`
	CheckOutAt     *time.Time `json:"check_out_at"`
	Checkpoint     string     `gorm:"size:50" json:"checkpoint"`
	ExpireMarkedAt *time.Time `json:"expire_marked_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// utcPtr 把可空时刻转到 UTC。
func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

// BeforeSave 入库前把全部时间字段归一化为 UTC：
// SQLite 驱动按 RFC3339 字符串存储，带时区偏移（如 +08:00）的串与 UTC 的串无法按字符串比较出正确先后；
// 统一为 UTC 后，重叠/到期等时间范围条件在 SQLite 与 MySQL 上结果一致。
func (p *VisitorPass) BeforeSave(tx *gorm.DB) error {
	p.StartTime = p.StartTime.UTC()
	p.EndTime = p.EndTime.UTC()
	p.ReviewedAt = utcPtr(p.ReviewedAt)
	p.CheckInAt = utcPtr(p.CheckInAt)
	p.CheckOutAt = utcPtr(p.CheckOutAt)
	p.ExpireMarkedAt = utcPtr(p.ExpireMarkedAt)
	if !p.CreatedAt.IsZero() {
		p.CreatedAt = p.CreatedAt.UTC()
	}
	if !p.UpdatedAt.IsZero() {
		p.UpdatedAt = p.UpdatedAt.UTC()
	}
	return nil
}

// VisitorEvent 访客凭证全生命周期留痕：取消、审核、进出与逾期各写一条。
// ActorID 为系统动作（如定时逾期标记）时为 NULL。
type VisitorEvent struct {
	ID         uint        `gorm:"primaryKey" json:"id"`
	PassID     uint        `gorm:"index" json:"pass_id"`
	Pass       VisitorPass `gorm:"foreignKey:PassID" json:"pass,omitempty"`
	Action     string      `gorm:"index;size:40" json:"action"`
	ActorID    *uint       `json:"actor_id"`
	Actor      *User       `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
	FromStatus string      `gorm:"size:20" json:"from_status"`
	ToStatus   string      `gorm:"size:20" json:"to_status"`
	Detail     string      `json:"detail"`
	CreatedAt  time.Time   `json:"created_at"`
}

// BeforeSave 留痕时刻同样归一化为 UTC。
func (e *VisitorEvent) BeforeSave(tx *gorm.DB) error {
	if !e.CreatedAt.IsZero() {
		e.CreatedAt = e.CreatedAt.UTC()
	}
	return nil
}
