package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/application/service"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

func newExportSvc(
	events *mockEventRepository,
	races *mockRaceRepository,
	cps *mockCheckpointRepository,
	runners *mockRunnerRepository,
) *service.EventExportImportService {
	return service.NewEventExportImportService(events, races, cps, runners)
}

func TestEventExportImportService_Export(t *testing.T) {
	dist := 12.5
	events := &mockEventRepository{events: []entity.Event{
		{ID: 1, Name: "GDR 2026"},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{
		1: {ID: 1, EventID: 1, Name: "GDR"},
	}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 1, Code: "AS1", DisplayName: "Aid Station 1", DisplayOrder: 1, DistanceFromStart: &dist},
		2: {ID: 2, RaceID: 1, Code: "AS2", DisplayName: "Aid Station 2", DisplayOrder: 2},
	}}
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 101, FirstName: "Alice", LastName: "Smith", SortOrder: 1},
			{ID: 2, RaceID: 1, BibNumber: 102, FirstName: "Bob", LastName: "Jones", SortOrder: 2},
		},
	}

	svc := newExportSvc(events, races, cps, runners)
	payload, err := svc.Export(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, 1, payload.Version)
	assert.Equal(t, "GDR 2026", payload.Event.Name)
	require.Len(t, payload.Races, 1)

	race := payload.Races[0]
	assert.Equal(t, "GDR", race.Name)
	require.Len(t, race.Checkpoints, 2)
	require.Len(t, race.Runners, 2)

	// Checkpoints are sorted by DisplayOrder; find AS1 specifically
	var as1 portsvc.CheckpointExport
	for _, cp := range race.Checkpoints {
		if cp.Code == "AS1" {
			as1 = cp
		}
	}
	assert.Equal(t, "Aid Station 1", as1.DisplayName)
	assert.Equal(t, &dist, as1.DistanceFromStart)

	assert.Equal(t, 101, race.Runners[0].BibNumber)
	assert.Equal(t, "Alice", race.Runners[0].FirstName)
}

func TestEventExportImportService_Import(t *testing.T) {
	events := &mockEventRepository{}
	races := &mockRaceRepository{races: map[int]entity.Race{}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}
	runners := &mockRunnerRepository{}

	svc := newExportSvc(events, races, cps, runners)

	dist := 10.0
	payload := portsvc.EventExportPayload{
		Version: 1,
		Event:   portsvc.EventExportInfo{Name: "Imported Event"},
		Races: []portsvc.RaceExportData{
			{
				Name: "50K",
				Checkpoints: []portsvc.CheckpointExport{
					{Code: "S", DisplayName: "Start", DisplayOrder: 1, DistanceFromStart: &dist},
				},
				Runners: []portsvc.RunnerExport{
					{BibNumber: 1, FirstName: "John", LastName: "Doe", SortOrder: 1},
				},
			},
		},
	}

	eventID, err := svc.Import(context.Background(), payload)
	require.NoError(t, err)
	assert.Equal(t, 1, eventID)

	// Verify the event was created
	ev, err := events.Get(context.Background(), eventID)
	require.NoError(t, err)
	assert.Equal(t, "Imported Event", ev.Name)

	// Verify runner was imported via BulkCreate
	assert.Len(t, runners.bulkCreated, 1)
	assert.Equal(t, 1, runners.bulkCreated[0].BibNumber)
}

func TestEventExportImportService_Export_EventNotFound(t *testing.T) {
	events := &mockEventRepository{}
	svc := newExportSvc(events, &mockRaceRepository{}, &mockCheckpointRepository{}, &mockRunnerRepository{})

	_, err := svc.Export(context.Background(), 999)
	assert.Error(t, err)
}
