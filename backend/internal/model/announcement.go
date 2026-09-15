package model

import "time"

type Announcement struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Category    string    `json:"category"`
	PublisherID uint      `json:"publisher_id"`
	Publisher   User      `json:"publisher"`
	PublishAt   time.Time `json:"publish_at"`
	Top         bool      `json:"top"`
	ReadCount   int       `json:"read_count"`
}
type AnnouncementRead struct {
	ID             uint `gorm:"primaryKey"`
	AnnouncementID uint `gorm:"uniqueIndex:idx_read"`
	UserID         uint `gorm:"uniqueIndex:idx_read"`
	CreatedAt      time.Time
}
