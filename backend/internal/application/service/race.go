package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type RaceService struct {
	repo portrepo.RaceRepository
}

func NewRaceService(repo portrepo.RaceRepository) *RaceService {
	return &RaceService{repo: repo}
}

var _ portsvc.RaceService = (*RaceService)(nil)

func (s *RaceService) List(ctx context.Context, eventID int) ([]entity.Race, error) {
	return s.repo.List(ctx, eventID)
}

func (s *RaceService) Get(ctx context.Context, id int) (entity.Race, error) {
	return s.repo.Get(ctx, id)
}

func (s *RaceService) Create(ctx context.Context, eventID int, name string) (entity.Race, error) {
	return s.repo.Create(ctx, eventID, name)
}

func (s *RaceService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
