package repository

import (
	"errors"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
)

type AnnouncementRepository struct{ DB *gorm.DB }

func NewAnnouncementRepository(db *gorm.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db}
}
func (r *AnnouncementRepository) Create(v *model.Announcement) error { return r.DB.Create(v).Error }
func (r *AnnouncementRepository) List() (out []model.Announcement, e error) {
	e = r.DB.Preload("Publisher").Order("top desc, publish_at desc").Find(&out).Error
	return
}
func (r *AnnouncementRepository) ByID(id uint) (v model.Announcement, e error) {
	e = r.DB.Preload("Publisher").First(&v, id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = ErrNotFound
	}
	return
}
func (r *AnnouncementRepository) MarkRead(id, uid uint) error {
	var rcd model.AnnouncementRead
	if e := r.DB.Where("announcement_id=? AND user_id=?", id, uid).First(&rcd).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		if e = r.DB.Create(&model.AnnouncementRead{AnnouncementID: id, UserID: uid}).Error; e != nil {
			return e
		}
		return r.DB.Model(&model.Announcement{}).Where("id=?", id).UpdateColumn("read_count", gorm.Expr("read_count + 1")).Error
	}
	return nil
}
