package model

import "time"

type Repair struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	User        User      `json:"user"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Images      string    `json:"images"`
	Status      string    `gorm:"index;size:20" json:"status"`
	HandlerID   *uint     `json:"handler_id"`
	Handler     *User     `gorm:"foreignKey:HandlerID" json:"handler,omitempty"`
	Rating      int       `json:"rating"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
