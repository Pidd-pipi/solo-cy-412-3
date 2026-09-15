package service

import (
	"fmt"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"github.com/smartestate/smartestate/internal/util"
	"log/slog"
)

type UserService struct {
	repo   *repository.UserRepository
	logger *slog.Logger
}

func NewUserService(r *repository.UserRepository, l *slog.Logger) *UserService {
	return &UserService{r, l}
}
func (s *UserService) Login(phone, password string) (model.User, error) {
	u, e := s.repo.ByPhone(phone)
	if e != nil {
		return u, fmt.Errorf("User[phone=%s] login failed: %w", phone, e)
	}
	if e = util.CheckPassword(u.PasswordHash, password); e != nil {
		return u, fmt.Errorf("User[phone=%s] password failed: %w", phone, e)
	}
	return u, nil
}
func (s *UserService) Me(id uint) (model.User, error) { return s.repo.ByID(id) }
func (s *UserService) Update(id uint, nick, avatar, building, unit, room string) (model.User, error) {
	u, e := s.repo.ByID(id)
	if e != nil {
		return u, fmt.Errorf("User[id=%d] profile failed: %w", id, e)
	}
	u.Nickname = nick
	u.Avatar = avatar
	u.Building = building
	u.Unit = unit
	u.Room = room
	e = s.repo.Update(&u)
	return u, e
}
func (s *UserService) Staff() ([]model.User, error) { return s.repo.ListStaff() }
