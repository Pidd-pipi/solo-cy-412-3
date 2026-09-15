package model

import "time"

type Payment struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"index" json:"user_id"`
	User      User       `json:"user"`
	FeeType   string     `json:"fee_type"`
	Amount    float64    `json:"amount"`
	Month     string     `json:"month"`
	Status    string     `gorm:"index;size:20" json:"status"`
	PaidAt    *time.Time `json:"paid_at"`
	CreatedAt time.Time  `json:"created_at"`
}
