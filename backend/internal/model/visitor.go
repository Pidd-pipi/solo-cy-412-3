package model

import "time"

// VisitorPass 访客通行凭证。同一访客（手机号）在同一楼栋的重叠时段只允许存在一张有效凭证。
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

// VisitorEvent 访客凭证全生命周期留痕：取消、审核、进出与逾期各写一条。
type VisitorEvent struct {
	ID         uint        `gorm:"primaryKey" json:"id"`
	PassID     uint        `gorm:"index" json:"pass_id"`
	Pass       VisitorPass `gorm:"foreignKey:PassID" json:"pass,omitempty"`
	Action     string      `gorm:"index;size:40" json:"action"`
	ActorID    uint        `json:"actor_id"`
	Actor      *User       `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
	FromStatus string      `gorm:"size:20" json:"from_status"`
	ToStatus   string      `gorm:"size:20" json:"to_status"`
	Detail     string      `json:"detail"`
	CreatedAt  time.Time   `json:"created_at"`
}
