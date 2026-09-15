package repository

import (
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
)

type OperationLogRepository struct{ DB *gorm.DB }

func NewOperationLogRepository(db *gorm.DB) *OperationLogRepository {
	return &OperationLogRepository{db}
}
func (r *OperationLogRepository) Create(v *model.OperationLog) error { return r.DB.Create(v).Error }

// CreateTx 在调用方事务/连接内写操作日志，保证与业务变更同提交。
func (r *OperationLogRepository) CreateTx(v *model.OperationLog, tx *gorm.DB) error {
	if tx != nil {
		return tx.Create(v).Error
	}
	return r.DB.Create(v).Error
}
func (r *OperationLogRepository) List() (out []model.OperationLog, e error) {
	e = r.DB.Order("created_at desc").Limit(100).Find(&out).Error
	return
}
