package service

import (
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"log/slog"
)

type OperationLogService struct {
	repo   *repository.OperationLogRepository
	logger *slog.Logger
}

func NewOperationLogService(r *repository.OperationLogRepository, l *slog.Logger) *OperationLogService {
	return &OperationLogService{r, l}
}
func (s *OperationLogService) Add(uid uint, action, detail string) {
	if e := s.repo.Create(&model.OperationLog{UserID: uid, Action: action, Detail: detail}); e != nil {
		s.logger.Error("write operation log", "error", e)
	}
}
func (s *OperationLogService) List() ([]model.OperationLog, error) { return s.repo.List() }
