package service

import (
	"context"
	"fmt"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type CheckpointService struct {
	checkpoints portrepo.CheckpointRepository
	races       portrepo.RaceRepository
}

func NewCheckpointService(checkpoints portrepo.CheckpointRepository, races portrepo.RaceRepository) *CheckpointService {
	return &CheckpointService{checkpoints: checkpoints, races: races}
}

var _ portsvc.CheckpointService = (*CheckpointService)(nil)

func (s *CheckpointService) List(ctx context.Context, raceID int) ([]entity.Checkpoint, error) {
	return s.checkpoints.List(ctx, raceID)
}

func (s *CheckpointService) Get(ctx context.Context, id int) (entity.Checkpoint, error) {
	return s.checkpoints.Get(ctx, id)
}

func (s *CheckpointService) Create(ctx context.Context, cp entity.Checkpoint) (entity.Checkpoint, error) {
	if cp.DisplayOrder == 0 {
		existing, err := s.checkpoints.List(ctx, cp.RaceID)
		if err != nil {
			return entity.Checkpoint{}, fmt.Errorf("listing checkpoints for order: %w", err)
		}
		cp.DisplayOrder = len(existing) + 1
	}
	return s.checkpoints.Create(ctx, cp)
}

func (s *CheckpointService) Update(ctx context.Context, id int, code, displayName string, distance *float64) (entity.Checkpoint, error) {
	cp, err := s.checkpoints.Get(ctx, id)
	if err != nil {
		return entity.Checkpoint{}, fmt.Errorf("getting checkpoint: %w", err)
	}
	race, err := s.races.Get(ctx, cp.RaceID)
	if err != nil {
		return entity.Checkpoint{}, fmt.Errorf("getting race: %w", err)
	}
	if race.OrderLocked {
		return entity.Checkpoint{}, fmt.Errorf("race %d: %w", cp.RaceID, domain.ErrLocked)
	}
	cp.Code = code
	cp.DisplayName = displayName
	cp.DistanceFromStart = distance
	return s.checkpoints.Update(ctx, cp)
}

func (s *CheckpointService) Delete(ctx context.Context, id int) error {
	cp, err := s.checkpoints.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("getting checkpoint: %w", err)
	}
	race, err := s.races.Get(ctx, cp.RaceID)
	if err != nil {
		return fmt.Errorf("getting race: %w", err)
	}
	if race.OrderLocked {
		return fmt.Errorf("race %d: %w", cp.RaceID, domain.ErrLocked)
	}
	return s.checkpoints.Delete(ctx, id)
}

func (s *CheckpointService) Reorder(ctx context.Context, raceID int, orderedIDs []int) error {
	race, err := s.races.Get(ctx, raceID)
	if err != nil {
		return fmt.Errorf("getting race: %w", err)
	}
	if race.OrderLocked {
		return fmt.Errorf("race %d: %w", raceID, domain.ErrLocked)
	}
	return s.checkpoints.Reorder(ctx, raceID, orderedIDs)
}
