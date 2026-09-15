package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Phone        string    `gorm:"uniqueIndex;size:30" json:"phone"`
	PasswordHash string    `json:"-"`
	Nickname     string    `json:"nickname"`
	Avatar       string    `json:"avatar"`
	Role         string    `gorm:"index;size:20" json:"role"`
	Building     string    `json:"building"`
	Unit         string    `json:"unit"`
	Room         string    `json:"room"`
	CreatedAt    time.Time `json:"created_at"`
}
