package service_test

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

// --- mockRunnerRepository ---

type mockRunnerRepository struct {
	runners         []entity.Runner
	updateStatus    func(id int, status entity.RunnerStatus)
	bulkCreated     []entity.Runner
	maxSortOrder    int
	listErr         error
	bulkCreateErr   error
	updateStatusErr error
	maxSortOrderErr error
}

func (m *mockRunnerRepository) List(_ context.Context, raceID int) ([]entity.Runner, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []entity.Runner
	for _, r := range m.runners {
		if r.RaceID == raceID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRunnerRepository) Get(_ context.Context, id int) (entity.Runner, error) {
	for _, r := range m.runners {
		if r.ID == id {
			return r, nil
		}
	}
	return entity.Runner{}, domain.ErrNotFound
}

func (m *mockRunnerRepository) GetByBibInEvent(_ context.Context, eventID, bibNumber int) (entity.Runner, error) {
	for _, r := range m.runners {
		if r.BibNumber == bibNumber {
			return r, nil
		}
	}
	return entity.Runner{}, domain.ErrNotFound
}

func (m *mockRunnerRepository) BulkCreate(_ context.Context, runners []entity.Runner) error {
	if m.bulkCreateErr != nil {
		return m.bulkCreateErr
	}
	m.bulkCreated = append(m.bulkCreated, runners...)
	m.runners = append(m.runners, runners...)
	return nil
}

func (m *mockRunnerRepository) UpdateStatus(_ context.Context, id int, status entity.RunnerStatus) error {
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	if m.updateStatus != nil {
		m.updateStatus(id, status)
	}
	for i := range m.runners {
		if m.runners[i].ID == id {
			m.runners[i].Status = status
		}
	}
	return nil
}

func (m *mockRunnerRepository) MaxSortOrder(_ context.Context, raceID int) (int, error) {
	return m.maxSortOrder, m.maxSortOrderErr
}

// --- mockRaceRepository ---

type mockRaceRepository struct {
	races      map[int]entity.Race
	lockedRace int
}

func (m *mockRaceRepository) List(_ context.Context, eventID int) ([]entity.Race, error) {
	var out []entity.Race
	for _, r := range m.races {
		if r.EventID == eventID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRaceRepository) Get(_ context.Context, id int) (entity.Race, error) {
	if r, ok := m.races[id]; ok {
		return r, nil
	}
	return entity.Race{}, domain.ErrNotFound
}

func (m *mockRaceRepository) Create(_ context.Context, eventID int, name string) (entity.Race, error) {
	return entity.Race{EventID: eventID, Name: name}, nil
}

func (m *mockRaceRepository) LockRoster(_ context.Context, id int) error {
	m.lockedRace = id
	if r, ok := m.races[id]; ok {
		r.RosterLocked = true
		m.races[id] = r
	}
	return nil
}

func (m *mockRaceRepository) LockOrder(_ context.Context, id int) error { return nil }
func (m *mockRaceRepository) Delete(_ context.Context, id int) error    { return nil }

// --- mockCheckpointLogRepository ---

type mockCheckpointLogRepository struct {
	logs      []entity.CheckpointLog
	created   []entity.CheckpointLog
	nextID    int
	existsErr error
	createErr error
	listErr   error
}

func (m *mockCheckpointLogRepository) Create(_ context.Context, log entity.CheckpointLog) (entity.CheckpointLog, error) {
	if m.createErr != nil {
		return entity.CheckpointLog{}, m.createErr
	}
	m.nextID++
	log.ID = m.nextID
	m.logs = append(m.logs, log)
	m.created = append(m.created, log)
	return log, nil
}

func (m *mockCheckpointLogRepository) ExistsByRunnerAndCheckpoint(_ context.Context, runnerID, checkpointID int) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	for _, l := range m.logs {
		if l.RunnerID == runnerID && l.CheckpointID == checkpointID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockCheckpointLogRepository) ListByRaceAndCheckpoint(_ context.Context, raceID, checkpointID int) ([]entity.CheckpointLog, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []entity.CheckpointLog
	for _, l := range m.logs {
		if l.CheckpointID == checkpointID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *mockCheckpointLogRepository) ListByRace(_ context.Context, raceID int) ([]entity.CheckpointLog, error) {
	return m.logs, nil
}

// --- mockActiveSessionRepository ---

type mockActiveSessionRepository struct {
	session entity.ActiveSession
	getErr  error
}

func (m *mockActiveSessionRepository) Get(_ context.Context) (entity.ActiveSession, error) {
	return m.session, m.getErr
}

func (m *mockActiveSessionRepository) SetEvent(_ context.Context, eventID int) error {
	m.session.EventID = &eventID
	return nil
}

func (m *mockActiveSessionRepository) SetCheckpoint(_ context.Context, raceID, checkpointID int) error {
	for i, sc := range m.session.Checkpoints {
		if sc.RaceID == raceID {
			m.session.Checkpoints[i].CheckpointID = checkpointID
			return nil
		}
	}
	m.session.Checkpoints = append(m.session.Checkpoints, entity.ActiveSessionCheckpoint{
		RaceID:       raceID,
		CheckpointID: checkpointID,
	})
	return nil
}

func (m *mockActiveSessionRepository) ClearCheckpoint(_ context.Context, raceID int) error {
	filtered := m.session.Checkpoints[:0]
	for _, sc := range m.session.Checkpoints {
		if sc.RaceID != raceID {
			filtered = append(filtered, sc)
		}
	}
	m.session.Checkpoints = filtered
	return nil
}

// --- mockCheckpointRepository ---

type mockCheckpointRepository struct {
	checkpoints map[int]entity.Checkpoint
	getErr      error
	listErr     error
}

func (m *mockCheckpointRepository) List(_ context.Context, raceID int) ([]entity.Checkpoint, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []entity.Checkpoint
	for _, cp := range m.checkpoints {
		if cp.RaceID == raceID {
			out = append(out, cp)
		}
	}
	return out, nil
}

func (m *mockCheckpointRepository) Get(_ context.Context, id int) (entity.Checkpoint, error) {
	if m.getErr != nil {
		return entity.Checkpoint{}, m.getErr
	}
	if cp, ok := m.checkpoints[id]; ok {
		return cp, nil
	}
	return entity.Checkpoint{}, domain.ErrNotFound
}

func (m *mockCheckpointRepository) Create(_ context.Context, cp entity.Checkpoint) (entity.Checkpoint, error) {
	return cp, nil
}

func (m *mockCheckpointRepository) Update(_ context.Context, cp entity.Checkpoint) (entity.Checkpoint, error) {
	return cp, nil
}

func (m *mockCheckpointRepository) Delete(_ context.Context, id int) error {
	return nil
}

func (m *mockCheckpointRepository) Reorder(_ context.Context, raceID int, orderedIDs []int) error {
	return nil
}
