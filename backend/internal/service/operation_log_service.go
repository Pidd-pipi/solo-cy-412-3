package service

import (
	"log/slog"

	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"gorm.io/gorm"
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

// AddTx 在调用方事务内写操作日志，与业务变更同一事务提交。
func (s *OperationLogService) AddTx(uid uint, action, detail string, tx *gorm.DB) error {
	return s.repo.CreateTx(&model.OperationLog{UserID: uid, Action: action, Detail: detail}, tx)
}

func (s *OperationLogService) List() ([]model.OperationLog, error) { return s.repo.List() }
