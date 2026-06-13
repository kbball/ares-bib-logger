package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type RosterRow struct {
	BibNumber int
	FirstName string
	LastName  string
}

type RunnerService interface {
	ImportRoster(ctx context.Context, raceID int, rows []RosterRow) error
	// TransferRace marks a runner MOVED and appends them to the target race.
	TransferRace(ctx context.Context, bibNumber, fromRaceID, toRaceID int) error
	ListByRace(ctx context.Context, raceID int) ([]entity.Runner, error)
}
