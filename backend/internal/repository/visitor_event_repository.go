package repository

import (
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
)

type VisitorEventRepository struct{ DB *gorm.DB }

func NewVisitorEventRepository(db *gorm.DB) *VisitorEventRepository {
	return &VisitorEventRepository{db}
}

func (r *VisitorEventRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.DB
}

// Create 写入一条访客生命周期留痕；可加入 service 事务。
func (r *VisitorEventRepository) Create(v *model.VisitorEvent, tx *gorm.DB) error {
	return r.tx(tx).Create(v).Error
}

// ListByPass 返回某张凭证的全部留痕（取消、审核、进出、逾期），时间正序。
func (r *VisitorEventRepository) ListByPass(passID uint) (out []model.VisitorEvent, e error) {
	e = r.DB.Preload("Actor").Where("pass_id = ?", passID).Order("created_at asc").Find(&out).Error
	return
}

// ListRecent 门岗记录 / 物业工作台使用的最近留痕。
func (r *VisitorEventRepository) ListRecent(limit int) (out []model.VisitorEvent, e error) {
	if limit <= 0 {
		limit = 50
	}
	e = r.DB.Preload("Actor").Preload("Pass").Order("created_at desc").Limit(limit).Find(&out).Error
	return
}
