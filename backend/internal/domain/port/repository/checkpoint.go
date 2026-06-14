package repository

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type CheckpointRepository interface {
	List(ctx context.Context, raceID int) ([]entity.Checkpoint, error)
	Get(ctx context.Context, id int) (entity.Checkpoint, error)
	Create(ctx context.Context, cp entity.Checkpoint) (entity.Checkpoint, error)
	Update(ctx context.Context, cp entity.Checkpoint) (entity.Checkpoint, error)
	Delete(ctx context.Context, id int) error
	// Reorder updates display_order for each checkpoint; orderedIDs is the full ordered slice.
	Reorder(ctx context.Context, raceID int, orderedIDs []int) error
}
