package repository

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type RunnerRepository interface {
	List(ctx context.Context, raceID int) ([]entity.Runner, error)
	Get(ctx context.Context, id int) (entity.Runner, error)
	// GetByBibInEvent looks across all races in an event — needed for multi-race events.
	GetByBibInEvent(ctx context.Context, eventID, bibNumber int) (entity.Runner, error)
	BulkCreate(ctx context.Context, runners []entity.Runner) error
	UpdateStatus(ctx context.Context, id int, status entity.RunnerStatus) error
	MaxSortOrder(ctx context.Context, raceID int) (int, error)
}
