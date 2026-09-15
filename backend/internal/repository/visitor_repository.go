package repository

import (
	"errors"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
	"time"
)

type VisitorPassRepository struct{ DB *gorm.DB }

func NewVisitorPassRepository(db *gorm.DB) *VisitorPassRepository { return &VisitorPassRepository{db} }

// tx 允许在 service 事务中复用同一连接；传入 nil 时使用默认连接。
func (r *VisitorPassRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.DB
}

func (r *VisitorPassRepository) Create(v *model.VisitorPass, tx *gorm.DB) error {
	return r.tx(tx).Create(v).Error
}
func (r *VisitorPassRepository) Update(v *model.VisitorPass, tx *gorm.DB) error {
	return r.tx(tx).Save(v).Error
}

func (r *VisitorPassRepository) preload(q *gorm.DB) *gorm.DB {
	return q.Preload("Resident").Preload("Reviewer")
}

// List 物业/门岗按状态、楼栋筛选全部凭证；residentID>0 时仅返回该业主登记的凭证。
func (r *VisitorPassRepository) List(residentID uint, status, building string) (out []model.VisitorPass, e error) {
	q := r.preload(r.DB.Model(&model.VisitorPass{})).Order("created_at desc")
	if residentID > 0 {
		q = q.Where("resident_id = ?", residentID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if building != "" {
		q = q.Where("building = ?", building)
	}
	e = q.Find(&out).Error
	return
}

func (r *VisitorPassRepository) ByID(id uint, tx *gorm.DB) (v model.VisitorPass, e error) {
	e = r.preload(r.tx(tx)).First(&v, id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = ErrNotFound
	}
	return
}
func (r *VisitorPassRepository) ByPassNo(passNo string, tx *gorm.DB) (v model.VisitorPass, e error) {
	e = r.preload(r.tx(tx)).Where("pass_no = ?", passNo).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = ErrNotFound
	}
	return
}

// FindOverlap 同一访客手机号、同一楼栋在 [start,end) 内已存在有效凭证（待审核/已通过/在场）即视为冲突。
// 重叠判定：existing.start < end AND existing.end > start。
func (r *VisitorPassRepository) FindOverlap(phone, building string, start, end time.Time, excludeID uint, tx *gorm.DB) (int64, error) {
	var n int64
	q := r.tx(tx).Model(&model.VisitorPass{}).
		Where("visitor_phone = ? AND building = ?", phone, building).
		Where("status IN ?", constants.ActivePassStatuses).
		Where("start_time < ? AND end_time > ?", end, start)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	e := q.Count(&n).Error
	return n, e
}

// CountInBuilding 某楼栋当前在场访客数（实时容量占用）。
func (r *VisitorPassRepository) CountInBuilding(building string, at time.Time, tx *gorm.DB) (int64, error) {
	var n int64
	e := r.tx(tx).Model(&model.VisitorPass{}).
		Where("building = ? AND status = ?", building, constants.PassStatusCheckedIn).
		Count(&n).Error
	return n, e
}

// ExpiredDue 已过结束时段但仍未离场（已通过未到门 / 在场超时）的凭证，供定时兜底任务标记逾期。
func (r *VisitorPassRepository) ExpiredDue(now time.Time) (out []model.VisitorPass, e error) {
	e = r.preload(r.DB).
		Where("status IN ?", []string{constants.PassStatusApproved, constants.PassStatusCheckedIn}).
		Where("end_time < ?", now).
		Find(&out).Error
	return
}

// CountPending 待审核凭证数量（物业工作台统计）。
func (r *VisitorPassRepository) CountPending() (int64, error) {
	var n int64
	e := r.DB.Model(&model.VisitorPass{}).Where("status = ?", constants.PassStatusPending).Count(&n).Error
	return n, e
}
