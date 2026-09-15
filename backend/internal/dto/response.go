package dto

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
type PageQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// PassUserView 凭证里嵌套的用户简要信息。
type PassUserView struct {
	ID       uint   `json:"id"`
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
}

// PassView 访客凭证响应：所有时刻统一格式化为社区时区钟面串（YYYY-MM-DD HH:mm），
// 业主、物业、门岗无论浏览器/服务器时区如何，都看到同一本地时刻；旧记录按原值展示。
type PassView struct {
	ID             uint          `json:"id"`
	PassNo         string        `json:"pass_no"`
	ResidentID     uint          `json:"resident_id"`
	Resident       PassUserView  `json:"resident"`
	VisitorName    string        `json:"visitor_name"`
	VisitorPhone   string        `json:"visitor_phone"`
	Building       string        `json:"building"`
	Reason         string        `json:"reason"`
	StartTime      string        `json:"start_time"`
	EndTime        string        `json:"end_time"`
	Status         string        `json:"status"`
	ReviewerID     *uint         `json:"reviewer_id"`
	Reviewer       *PassUserView `json:"reviewer,omitempty"`
	ReviewRemark   string        `json:"review_remark"`
	ReviewedAt     string        `json:"reviewed_at"`
	CheckInAt      string        `json:"check_in_at"`
	CheckOutAt     string        `json:"check_out_at"`
	Checkpoint     string        `json:"checkpoint"`
	ExpireMarkedAt string        `json:"expire_marked_at"`
	CreatedAt      string        `json:"created_at"`
}

// PassEventView 凭证留痕响应。
type PassEventView struct {
	ID         uint          `json:"id"`
	PassID     uint          `json:"pass_id"`
	Action     string        `json:"action"`
	ActorID    *uint         `json:"actor_id"`
	Actor      *PassUserView `json:"actor,omitempty"`
	FromStatus string        `json:"from_status"`
	ToStatus   string        `json:"to_status"`
	Detail     string        `json:"detail"`
	CreatedAt  string        `json:"created_at"`
}
