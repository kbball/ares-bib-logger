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
	// AddRunner appends a single runner to the bottom of a race's roster
	// (sort_order = max existing + 1) — for a late registration not covered
	// by the initial roster import. Unlike ImportRoster, this is not gated
	// by RosterLocked and never locks the roster.
	AddRunner(ctx context.Context, raceID, bibNumber int, firstName, lastName string) error
}
