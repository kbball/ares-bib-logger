package repository

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type CheckpointLogRepository interface {
	Create(ctx context.Context, log entity.CheckpointLog) (entity.CheckpointLog, error)
	// Upsert inserts a checkpoint log or overwrites the existing one for the same
	// (runner_id, checkpoint_id) pair. The bool return is true when the row was
	// newly inserted, false when an existing row was updated.
	Upsert(ctx context.Context, log entity.CheckpointLog) (entity.CheckpointLog, bool, error)
	ExistsByRunnerAndCheckpoint(ctx context.Context, runnerID, checkpointID int) (bool, error)
	ListByRaceAndCheckpoint(ctx context.Context, raceID, checkpointID int) ([]entity.CheckpointLog, error)
	ListByRace(ctx context.Context, raceID int) ([]entity.CheckpointLog, error)
}
