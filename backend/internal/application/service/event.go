package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type EventService struct {
	repo portrepo.EventRepository
}

func NewEventService(repo portrepo.EventRepository) *EventService {
	return &EventService{repo: repo}
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
