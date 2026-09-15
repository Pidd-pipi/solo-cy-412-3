package repository

import (
	"errors"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct{ DB *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db} }
func (r *UserRepository) ByID(id uint) (model.User, error) {
	var v model.User
	e := r.DB.First(&v, id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return v, ErrNotFound
	}
	return v, e
}
func (r *UserRepository) ByPhone(phone string) (model.User, error) {
	var v model.User
	e := r.DB.Where("phone = ?", phone).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return v, ErrNotFound
	}
	return v, e
}
func (r *UserRepository) Update(v *model.User) error { return r.DB.Save(v).Error }
func (r *UserRepository) ListStaff() (out []model.User, e error) {
	e = r.DB.Where("role IN ?", []string{"staff", "admin"}).Find(&out).Error
	return
}
