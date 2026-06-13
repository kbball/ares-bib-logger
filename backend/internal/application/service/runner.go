package service

import (
	"context"
	"fmt"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type RunnerService struct {
	runners portrepo.RunnerRepository
	races   portrepo.RaceRepository
}

func NewRunnerService(runners portrepo.RunnerRepository, races portrepo.RaceRepository) *RunnerService {
	return &RunnerService{runners: runners, races: races}
}

var _ portsvc.RunnerService = (*RunnerService)(nil)

func (s *RunnerService) ImportRoster(ctx context.Context, raceID int, rows []portsvc.RosterRow) error {
	race, err := s.races.Get(ctx, raceID)
	if err != nil {
		return fmt.Errorf("getting race: %w", err)
	}
	if race.RosterLocked {
		return fmt.Errorf("race %d: %w", raceID, errRosterLocked)
	}

	runners := make([]entity.Runner, len(rows))
	for i, row := range rows {
		runners[i] = entity.Runner{
			RaceID:    raceID,
			BibNumber: row.BibNumber,
			FirstName: row.FirstName,
			LastName:  row.LastName,
			SortOrder: i + 1,
			Status:    entity.StatusUnknown,
		}
	}

	if err := s.runners.BulkCreate(ctx, runners); err != nil {
		return fmt.Errorf("creating runners: %w", err)
	}

	return s.races.LockRoster(ctx, raceID)
}

func (s *RunnerService) TransferRace(ctx context.Context, bibNumber, fromRaceID, toRaceID int) error {
	all, err := s.runners.List(ctx, fromRaceID)
	if err != nil {
		return fmt.Errorf("listing runners in race %d: %w", fromRaceID, err)
	}

	var found *entity.Runner
	for i := range all {
		if all[i].BibNumber == bibNumber {
			found = &all[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("bib %d not found in race %d", bibNumber, fromRaceID)
	}

	if err := s.runners.UpdateStatus(ctx, found.ID, entity.StatusMoved); err != nil {
		return fmt.Errorf("marking runner moved: %w", err)
	}

	max, err := s.runners.MaxSortOrder(ctx, toRaceID)
	if err != nil {
		return fmt.Errorf("getting max sort order: %w", err)
	}

	return s.runners.BulkCreate(ctx, []entity.Runner{{
		RaceID:    toRaceID,
		BibNumber: found.BibNumber,
		FirstName: found.FirstName,
		LastName:  found.LastName,
		SortOrder: max + 1,
		Status:    entity.StatusActive,
	}})
}

func (s *RunnerService) ListByRace(ctx context.Context, raceID int) ([]entity.Runner, error) {
	return s.runners.List(ctx, raceID)
}

var errRosterLocked = fmt.Errorf("roster is already locked")
