package handler_test

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

// noopPublisher satisfies sse.Publisher for tests that don't assert SSE output.
type noopPublisher struct{}

func (noopPublisher) Publish(_ string, _ any) {}

// --- mock services ---

type mockEventService struct {
	events []entity.Event
	err    error
}

func (m *mockEventService) List(_ context.Context) ([]entity.Event, error) { return m.events, m.err }
func (m *mockEventService) Get(_ context.Context, id int) (entity.Event, error) {
	for _, e := range m.events {
		if e.ID == id {
			return e, m.err
		}
	}
	return entity.Event{}, m.err
}
func (m *mockEventService) Create(_ context.Context, name string) (entity.Event, error) {
	e := entity.Event{ID: 1, Name: name}
	return e, m.err
}
func (m *mockEventService) Archive(_ context.Context, id int) error { return m.err }

type mockRaceService struct {
	races []entity.Race
	err   error
}

func (m *mockRaceService) List(_ context.Context, eventID int) ([]entity.Race, error) {
	return m.races, m.err
}
func (m *mockRaceService) Get(_ context.Context, id int) (entity.Race, error) {
	for _, r := range m.races {
		if r.ID == id {
			return r, m.err
		}
	}
	return entity.Race{}, m.err
}
func (m *mockRaceService) Create(_ context.Context, eventID int, name string) (entity.Race, error) {
	return entity.Race{ID: 1, EventID: eventID, Name: name}, m.err
}
func (m *mockRaceService) Delete(_ context.Context, id int) error   { return m.err }
func (m *mockRaceService) LockOrder(_ context.Context, id int) error { return m.err }

type mockCheckpointService struct {
	checkpoints []entity.Checkpoint
	err         error
}

func (m *mockCheckpointService) List(_ context.Context, raceID int) ([]entity.Checkpoint, error) {
	return m.checkpoints, m.err
}
func (m *mockCheckpointService) Get(_ context.Context, id int) (entity.Checkpoint, error) {
	return entity.Checkpoint{}, m.err
}
func (m *mockCheckpointService) Create(_ context.Context, cp entity.Checkpoint) (entity.Checkpoint, error) {
	cp.ID = 1
	return cp, m.err
}
func (m *mockCheckpointService) Update(_ context.Context, id int, code, displayName string) (entity.Checkpoint, error) {
	return entity.Checkpoint{ID: id, Code: code, DisplayName: displayName}, m.err
}
func (m *mockCheckpointService) Delete(_ context.Context, id int) error { return m.err }
func (m *mockCheckpointService) Reorder(_ context.Context, raceID int, ids []int) error {
	return m.err
}

type mockRunnerService struct {
	runners      []entity.Runner
	importCalled bool
	err          error
}

func (m *mockRunnerService) ListByRace(_ context.Context, raceID int) ([]entity.Runner, error) {
	return m.runners, m.err
}
func (m *mockRunnerService) ImportRoster(_ context.Context, raceID int, rows []portsvc.RosterRow) error {
	m.importCalled = true
	return m.err
}
func (m *mockRunnerService) TransferRace(_ context.Context, bib, from, to int) error { return m.err }

type mockCheckpointLogService struct {
	result portsvc.LogBibResult
	err    error
}

func (m *mockCheckpointLogService) LogBib(_ context.Context, input portsvc.LogBibInput) (portsvc.LogBibResult, error) {
	return m.result, m.err
}
func (m *mockCheckpointLogService) LogStatus(_ context.Context, bib int, status entity.RunnerStatus) error {
	return m.err
}
func (m *mockCheckpointLogService) ListByRace(_ context.Context, raceID int) ([]entity.CheckpointLog, error) {
	return nil, m.err
}

type mockSessionService struct {
	session entity.ActiveSession
	err     error
}

func (m *mockSessionService) Get(_ context.Context) (entity.ActiveSession, error) {
	return m.session, m.err
}
func (m *mockSessionService) SetEvent(_ context.Context, id int) error            { return m.err }
func (m *mockSessionService) SetCheckpoint(_ context.Context, r, c int) error     { return m.err }
func (m *mockSessionService) ClearCheckpoint(_ context.Context, raceID int) error { return m.err }

type mockWinlinkService struct {
	exportText string
	importRes  portsvc.WinlinkImportResult
	err        error
}

func (m *mockWinlinkService) Export(_ context.Context, raceID int) (string, error) {
	return m.exportText, m.err
}
func (m *mockWinlinkService) Import(_ context.Context, raceID, checkpointID int, text string) (portsvc.WinlinkImportResult, error) {
	return m.importRes, m.err
}
