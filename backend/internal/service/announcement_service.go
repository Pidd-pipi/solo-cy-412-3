package service

import (
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"log/slog"
	"time"
)

type AnnouncementService struct {
	repo   *repository.AnnouncementRepository
	logger *slog.Logger
}

func NewAnnouncementService(r *repository.AnnouncementRepository, l *slog.Logger) *AnnouncementService {
	return &AnnouncementService{r, l}
}
func (s *AnnouncementService) Create(uid uint, title, content, category string, top bool) (model.Announcement, error) {
	v := model.Announcement{Title: title, Content: content, Category: category, PublisherID: uid, Top: top, PublishAt: time.Now()}
	e := s.repo.Create(&v)
	return v, e
}
func (s *AnnouncementService) List() ([]model.Announcement, error) { return s.repo.List() }
func (s *AnnouncementService) Detail(id, uid uint) (model.Announcement, error) {
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, e
	}
	if e = s.repo.MarkRead(id, uid); e != nil {
		return v, e
	}
	// Reload so the response immediately contains the incremented read count.
	return s.repo.ByID(id)
}
