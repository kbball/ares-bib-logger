package entity_test

import (
	"testing"
	"time"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

// RunnerStatus constants must match the values stored in Postgres.
func TestRunnerStatusConstants(t *testing.T) {
	assert.Equal(t, entity.RunnerStatus("UNKNOWN"), entity.StatusUnknown)
	assert.Equal(t, entity.RunnerStatus("ACTIVE"), entity.StatusActive)
	assert.Equal(t, entity.RunnerStatus("DNS"), entity.StatusDNS)
	assert.Equal(t, entity.RunnerStatus("DNF"), entity.StatusDNF)
	assert.Equal(t, entity.RunnerStatus("FINISHED"), entity.StatusFinished)
	assert.Equal(t, entity.RunnerStatus("MOVED"), entity.StatusMoved)
}

// LogSource constants must match the values stored in Postgres.
func TestLogSourceConstants(t *testing.T) {
	assert.Equal(t, entity.LogSource("MESHTASTIC"), entity.SourceMeshtastic)
	assert.Equal(t, entity.LogSource("MANUAL"), entity.SourceManual)
	assert.Equal(t, entity.LogSource("WINLINK_IMPORT"), entity.SourceWinlinkImport)
}

func TestRunner_Fields(t *testing.T) {
	now := time.Now()
	r := entity.Runner{
		ID:        1,
		RaceID:    2,
		BibNumber: 100,
		FirstName: "Alice",
		LastName:  "Smith",
		SortOrder: 3,
		Status:    entity.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	assert.Equal(t, 1, r.ID)
	assert.Equal(t, 100, r.BibNumber)
	assert.Equal(t, entity.StatusActive, r.Status)
}

func TestCheckpoint_NilDistance(t *testing.T) {
	cp := entity.Checkpoint{
		ID:           1,
		RaceID:       1,
		Code:         "AS1",
		DisplayName:  "Aid Station 1",
		DisplayOrder: 1,
	}
	assert.Nil(t, cp.DistanceFromStart)
}

func TestCheckpoint_WithDistance(t *testing.T) {
	dist := 10.5
	cp := entity.Checkpoint{
		ID:                1,
		RaceID:            1,
		Code:              "AS1",
		DisplayName:       "Aid Station 1",
		DisplayOrder:      1,
		DistanceFromStart: &dist,
	}
	assert.NotNil(t, cp.DistanceFromStart)
	assert.InDelta(t, 10.5, *cp.DistanceFromStart, 0.001)
}

func TestActiveSession_NoEvent(t *testing.T) {
	s := entity.ActiveSession{}
	assert.Nil(t, s.EventID)
	assert.Empty(t, s.Checkpoints)
}

func TestActiveSession_WithCheckpoints(t *testing.T) {
	eventID := 1
	s := entity.ActiveSession{
		EventID: &eventID,
		Checkpoints: []entity.ActiveSessionCheckpoint{
			{RaceID: 1, CheckpointID: 10},
			{RaceID: 2, CheckpointID: 20},
		},
	}
	assert.Equal(t, 1, *s.EventID)
	assert.Len(t, s.Checkpoints, 2)
	assert.Equal(t, 10, s.Checkpoints[0].CheckpointID)
}

func TestCheckpointLog_Sources(t *testing.T) {
	now := time.Now()
	log := entity.CheckpointLog{
		ID:           1,
		RunnerID:     1,
		CheckpointID: 1,
		RecordedAt:   now,
		Source:       entity.SourceManual,
		RawMessage:   "10:00",
		CreatedAt:    now,
	}
	assert.Equal(t, entity.SourceManual, log.Source)
	assert.Equal(t, "10:00", log.RawMessage)
}

func TestEvent_Fields(t *testing.T) {
	now := time.Now()
	e := entity.Event{
		ID:        1,
		Name:      "GA Death Race",
		Archived:  false,
		CreatedAt: now,
	}
	assert.Equal(t, "GA Death Race", e.Name)
	assert.False(t, e.Archived)
}

func TestRace_Fields(t *testing.T) {
	now := time.Now()
	r := entity.Race{
		ID:           1,
		EventID:      1,
		Name:         "GDR",
		RosterLocked: false,
		OrderLocked:  true,
		CreatedAt:    now,
	}
	assert.Equal(t, "GDR", r.Name)
	assert.False(t, r.RosterLocked)
	assert.True(t, r.OrderLocked)
}
