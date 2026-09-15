package service

import (
	"fmt"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"log/slog"
	"time"
)

type PaymentService struct {
	repo   *repository.PaymentRepository
	logger *slog.Logger
}

func NewPaymentService(r *repository.PaymentRepository, l *slog.Logger) *PaymentService {
	return &PaymentService{r, l}
}
func (s *PaymentService) Create(uid uint, fee string, amount float64, month string) (model.Payment, error) {
	v := model.Payment{UserID: uid, FeeType: fee, Amount: amount, Month: month, Status: "unpaid"}
	e := s.repo.Create(&v)
	return v, e
}
func (s *PaymentService) List(uid uint) ([]model.Payment, error) { return s.repo.List(uid) }
func (s *PaymentService) Pay(id, uid uint, role string) (model.Payment, error) {
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, e
	}
	if role == constants.UserRoleResident && v.UserID != uid {
		return v, fmt.Errorf("Payment[id=%d] pay forbidden: current user=%d owner=%d", id, uid, v.UserID)
	}
	if v.Status == "paid" {
		return v, nil
	}
	now := time.Now()
	v.Status = "paid"
	v.PaidAt = &now
	if e = s.repo.Update(&v); e != nil {
		return v, fmt.Errorf("Payment[id=%d] pay failed: %w", id, e)
	}
	return v, nil
}
func (s *PaymentService) MonthlyPaid() (float64, error) { return s.repo.MonthlyPaid() }
