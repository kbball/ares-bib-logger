package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type SessionService struct {
	repo portrepo.ActiveSessionRepository
}

func NewSessionService(repo portrepo.ActiveSessionRepository) *SessionService {
	return &SessionService{repo: repo}
}

var _ portsvc.SessionService = (*SessionService)(nil)

func (s *SessionService) Get(ctx context.Context) (entity.ActiveSession, error) {
	return s.repo.Get(ctx)
}

func (s *SessionService) SetEvent(ctx context.Context, eventID int) error {
	return s.repo.SetEvent(ctx, eventID)
}

func (s *SessionService) SetCheckpoint(ctx context.Context, raceID, checkpointID int) error {
	return s.repo.SetCheckpoint(ctx, raceID, checkpointID)
}

func (s *SessionService) ClearCheckpoint(ctx context.Context, raceID int) error {
	return s.repo.ClearCheckpoint(ctx, raceID)
}
