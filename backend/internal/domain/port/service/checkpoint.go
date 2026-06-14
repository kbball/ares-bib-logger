package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type CheckpointService interface {
	List(ctx context.Context, raceID int) ([]entity.Checkpoint, error)
	Get(ctx context.Context, id int) (entity.Checkpoint, error)
	Create(ctx context.Context, cp entity.Checkpoint) (entity.Checkpoint, error)
	// Update changes code and display_name; returns domain.ErrLocked if race order is locked.
	Update(ctx context.Context, id int, code, displayName string) (entity.Checkpoint, error)
	Delete(ctx context.Context, id int) error
	// Reorder returns domain.ErrLocked if the race's checkpoint order is locked.
	Reorder(ctx context.Context, raceID int, orderedIDs []int) error
}
