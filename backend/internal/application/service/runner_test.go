package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/application/service"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

func newRunnerSvc(runners *mockRunnerRepository, races *mockRaceRepository) *service.RunnerService {
	return service.NewRunnerService(runners, races)
}

func TestRunnerService_ImportRoster_Success(t *testing.T) {
	runners := &mockRunnerRepository{}
	races := &mockRaceRepository{
		races: map[int]entity.Race{
			1: {ID: 1, EventID: 10, Name: "GDR", RosterLocked: false},
		},
	}

	svc := newRunnerSvc(runners, races)
	err := svc.ImportRoster(context.Background(), 1, []portsvc.RosterRow{
		{BibNumber: 100, FirstName: "Alice", LastName: "Smith"},
		{BibNumber: 101, FirstName: "Bob", LastName: "Jones"},
		{BibNumber: 102, FirstName: "Carol", LastName: "Lee"},
	})

	require.NoError(t, err)
	assert.Len(t, runners.bulkCreated, 3)
	assert.Equal(t, 1, runners.bulkCreated[0].SortOrder)
	assert.Equal(t, 2, runners.bulkCreated[1].SortOrder)
	assert.Equal(t, 3, runners.bulkCreated[2].SortOrder)
	assert.Equal(t, "Alice", runners.bulkCreated[0].FirstName)
	assert.Equal(t, 1, races.lockedRace)
}

func TestRunnerService_ImportRoster_LockedReturnsError(t *testing.T) {
	runners := &mockRunnerRepository{}
	races := &mockRaceRepository{
		races: map[int]entity.Race{
			1: {ID: 1, RosterLocked: true},
		},
	}

	svc := newRunnerSvc(runners, races)
	err := svc.ImportRoster(context.Background(), 1, []portsvc.RosterRow{
		{BibNumber: 100, FirstName: "Alice", LastName: "Smith"},
	})

	assert.Error(t, err)
	assert.Empty(t, runners.bulkCreated)
}

func TestRunnerService_TransferRace_Success(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 5, RaceID: 1, BibNumber: 42, FirstName: "Alice", LastName: "Smith", SortOrder: 3},
		},
		maxSortOrder: 7,
	}
	races := &mockRaceRepository{
		races: map[int]entity.Race{
			1: {ID: 1}, 2: {ID: 2},
		},
	}

	svc := newRunnerSvc(runners, races)
	err := svc.TransferRace(context.Background(), 42, 1, 2)

	require.NoError(t, err)

	// Original runner should be MOVED
	assert.Equal(t, entity.StatusMoved, runners.runners[0].Status)

	// New runner created in race 2 with sort_order = maxSortOrder + 1
	require.Len(t, runners.bulkCreated, 1)
	created := runners.bulkCreated[0]
	assert.Equal(t, 2, created.RaceID)
	assert.Equal(t, 42, created.BibNumber)
	assert.Equal(t, 8, created.SortOrder)
	assert.Equal(t, entity.StatusActive, created.Status)
}

func TestRunnerService_TransferRace_BibNotFound(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 5, RaceID: 1, BibNumber: 42},
		},
	}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1}, 2: {ID: 2}}}

	svc := newRunnerSvc(runners, races)
	err := svc.TransferRace(context.Background(), 999, 1, 2)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "not found")
}

func TestRunnerService_ListByRace(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 10, BibNumber: 100},
			{ID: 2, RaceID: 10, BibNumber: 101},
			{ID: 3, RaceID: 99, BibNumber: 200},
		},
	}
	races := &mockRaceRepository{races: map[int]entity.Race{}}

	svc := newRunnerSvc(runners, races)
	result, err := svc.ListByRace(context.Background(), 10)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestRunnerService_ImportRoster_GetRaceError(t *testing.T) {
	svc := newRunnerSvc(&mockRunnerRepository{}, &mockRaceRepository{races: map[int]entity.Race{}})
	err := svc.ImportRoster(context.Background(), 999, []portsvc.RosterRow{{BibNumber: 1}})
	assert.Error(t, err)
}

func TestRunnerService_ImportRoster_BulkCreateError(t *testing.T) {
	dbErr := errors.New("db down")
	runners := &mockRunnerRepository{bulkCreateErr: dbErr}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, RosterLocked: false}}}

	svc := newRunnerSvc(runners, races)
	err := svc.ImportRoster(context.Background(), 1, []portsvc.RosterRow{{BibNumber: 100}})

	assert.ErrorContains(t, err, "creating runners")
}

func TestRunnerService_TransferRace_ListError(t *testing.T) {
	dbErr := errors.New("db down")
	runners := &mockRunnerRepository{listErr: dbErr}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1}, 2: {ID: 2}}}

	svc := newRunnerSvc(runners, races)
	err := svc.TransferRace(context.Background(), 42, 1, 2)

	assert.ErrorContains(t, err, "listing runners")
}

func TestRunnerService_TransferRace_UpdateStatusError(t *testing.T) {
	dbErr := errors.New("db down")
	runners := &mockRunnerRepository{
		runners:         []entity.Runner{{ID: 5, RaceID: 1, BibNumber: 42}},
		updateStatusErr: dbErr,
	}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1}, 2: {ID: 2}}}

	svc := newRunnerSvc(runners, races)
	err := svc.TransferRace(context.Background(), 42, 1, 2)

	assert.ErrorContains(t, err, "marking runner moved")
}

func TestRunnerService_TransferRace_MaxSortOrderError(t *testing.T) {
	dbErr := errors.New("db down")
	runners := &mockRunnerRepository{
		runners:         []entity.Runner{{ID: 5, RaceID: 1, BibNumber: 42}},
		maxSortOrderErr: dbErr,
	}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1}, 2: {ID: 2}}}

	svc := newRunnerSvc(runners, races)
	err := svc.TransferRace(context.Background(), 42, 1, 2)

	assert.ErrorContains(t, err, "getting max sort order")
}
