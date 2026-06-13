package repository

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type CheckpointLogRepository interface {
	Create(ctx context.Context, log entity.CheckpointLog) (entity.CheckpointLog, error)
	ExistsByRunnerAndCheckpoint(ctx context.Context, runnerID, checkpointID int) (bool, error)
	ListByRaceAndCheckpoint(ctx context.Context, raceID, checkpointID int) ([]entity.CheckpointLog, error)
	ListByRace(ctx context.Context, raceID int) ([]entity.CheckpointLog, error)
}
