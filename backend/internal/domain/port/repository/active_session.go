package repository

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type ActiveSessionRepository interface {
	Get(ctx context.Context) (entity.ActiveSession, error)
	SetEvent(ctx context.Context, eventID int) error
	SetCheckpoint(ctx context.Context, raceID, checkpointID int) error
	ClearCheckpoint(ctx context.Context, raceID int) error
}
