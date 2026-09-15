package repository

import (
	"errors"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
	"time"
)

type PaymentRepository struct{ DB *gorm.DB }

func NewPaymentRepository(db *gorm.DB) *PaymentRepository  { return &PaymentRepository{db} }
func (r *PaymentRepository) Create(v *model.Payment) error { return r.DB.Create(v).Error }
func (r *PaymentRepository) List(userID uint) (out []model.Payment, e error) {
	q := r.DB.Preload("User").Order("created_at desc")
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	e = q.Find(&out).Error
	return
}
func (r *PaymentRepository) ByID(id uint) (v model.Payment, e error) {
	e = r.DB.Preload("User").First(&v, id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = ErrNotFound
	}
	return
}
func (r *PaymentRepository) Update(v *model.Payment) error { return r.DB.Save(v).Error }
func (r *PaymentRepository) MonthlyPaid() (float64, error) {
	var amount float64
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0)
	e := r.DB.Model(&model.Payment{}).Select("COALESCE(SUM(amount),0)").Where("status = ? AND paid_at >= ? AND paid_at < ?", "paid", start, end).Scan(&amount).Error
	return amount, e
}
