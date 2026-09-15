package model

import "time"

type OperationLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}
type Role struct {
	ID   uint   `gorm:"primaryKey"`
	Code string `gorm:"uniqueIndex;size:64"`
	Name string
}
type Permission struct {
	ID   uint   `gorm:"primaryKey"`
	Code string `gorm:"uniqueIndex;size:64"`
	Name string
}
type RolePermission struct {
	RoleID       uint `gorm:"primaryKey"`
	PermissionID uint `gorm:"primaryKey"`
}
