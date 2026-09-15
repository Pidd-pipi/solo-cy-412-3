package dto

type LoginRequest struct {
	Phone    string `json:"phone" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" validate:"required,min=2,max=30"`
	Avatar   string `json:"avatar"`
	Building string `json:"building"`
	Unit     string `json:"unit"`
	Room     string `json:"room"`
}
type CreateRepairRequest struct {
	Title       string `json:"title" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"required,min=5"`
	Type        string `json:"type" validate:"required,oneof=水电 家具 公共设施 其他"`
	Images      string `json:"images"`
}
type AssignRepairRequest struct {
	HandlerID uint `json:"handler_id" validate:"required"`
}
type UpdateRepairStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending assigned processing done closed"`
	Rating int    `json:"rating" validate:"omitempty,min=1,max=5"`
}
type CreatePaymentRequest struct {
	UserID  uint    `json:"user_id" validate:"required"`
	FeeType string  `json:"fee_type" validate:"required"`
	Amount  float64 `json:"amount" validate:"required,gt=0"`
	Month   string  `json:"month" validate:"required,len=7"`
}
type CreateAnnouncementRequest struct {
	Title    string `json:"title" validate:"required,min=2"`
	Content  string `json:"content" validate:"required,min=5"`
	Category string `json:"category" validate:"required,oneof=通知 活动 紧急"`
	Top      bool   `json:"top"`
}
