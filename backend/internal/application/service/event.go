package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type EventService struct {
	repo        portrepo.EventRepository
	sessionRepo portrepo.ActiveSessionRepository
}

func NewEventService(repo portrepo.EventRepository, sessionRepo portrepo.ActiveSessionRepository) *EventService {
	return &EventService{repo: repo, sessionRepo: sessionRepo}
}

var _ portsvc.EventService = (*EventService)(nil)

func (s *EventService) List(ctx context.Context) ([]entity.Event, error) {
	return s.repo.List(ctx)
}

func (s *EventService) Get(ctx context.Context, id int) (entity.Event, error) {
	return s.repo.Get(ctx, id)
}

func (s *EventService) Create(ctx context.Context, name string) (entity.Event, error) {
	return s.repo.Create(ctx, name)
}

func (s *EventService) Archive(ctx context.Context, id int) error {
	if err := s.repo.Archive(ctx, id); err != nil {
		return err
	}

	sess, err := s.sessionRepo.Get(ctx)
	if err != nil {
		return err
	}
	if sess.EventID != nil && *sess.EventID == id {
		return s.sessionRepo.ClearEvent(ctx)
	}
	return nil
}

func (s *EventService) SetWinlinkBlankLineAfterHeader(ctx context.Context, id int, enabled bool) error {
	return s.repo.SetWinlinkBlankLineAfterHeader(ctx, id, enabled)
}
