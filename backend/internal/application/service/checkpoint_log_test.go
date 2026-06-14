package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/application/service"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

func newCheckpointLogSvc(
	runners *mockRunnerRepository,
	logs *mockCheckpointLogRepository,
	sess *mockActiveSessionRepository,
) *service.CheckpointLogService {
	return service.NewCheckpointLogService(runners, logs, sess)
}

func TestCheckpointLogService_LogBib_Success(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 10, RaceID: 5, BibNumber: 42, Status: entity.StatusActive}},
	}
	logs := &mockCheckpointLogRepository{}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 5, CheckpointID: 99}},
	}}

	svc := newCheckpointLogSvc(runners, logs, sess)
	result, err := svc.LogBib(context.Background(), portsvc.LogBibInput{
		BibNumber:  42,
		Source:     entity.SourceManual,
		RawMessage: "",
	})

	require.NoError(t, err)
	assert.False(t, result.IsDuplicate)
	assert.Equal(t, 10, result.Runner.ID)
	assert.Equal(t, 10, result.Log.RunnerID)
	assert.Equal(t, 99, result.Log.CheckpointID)
	assert.Equal(t, entity.SourceManual, result.Log.Source)
	assert.Len(t, logs.created, 1)
}

func TestCheckpointLogService_LogBib_Duplicate(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 10, RaceID: 5, BibNumber: 42}},
	}
	logs := &mockCheckpointLogRepository{
		logs: []entity.CheckpointLog{{RunnerID: 10, CheckpointID: 99}},
	}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 5, CheckpointID: 99}},
	}}

	svc := newCheckpointLogSvc(runners, logs, sess)
	result, err := svc.LogBib(context.Background(), portsvc.LogBibInput{BibNumber: 42})

	require.NoError(t, err)
	assert.True(t, result.IsDuplicate)
	assert.Empty(t, logs.created)
}

func TestCheckpointLogService_LogBib_NoActiveEvent(t *testing.T) {
	svc := newCheckpointLogSvc(
		&mockRunnerRepository{},
		&mockCheckpointLogRepository{},
		&mockActiveSessionRepository{session: entity.ActiveSession{}},
	)

	_, err := svc.LogBib(context.Background(), portsvc.LogBibInput{BibNumber: 42})

	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestCheckpointLogService_LogBib_UnknownBib(t *testing.T) {
	eventIDVal := 1
	svc := newCheckpointLogSvc(
		&mockRunnerRepository{},
		&mockCheckpointLogRepository{},
		&mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}},
	)

	_, err := svc.LogBib(context.Background(), portsvc.LogBibInput{BibNumber: 999})

	assert.ErrorContains(t, err, "999")
}

func TestCheckpointLogService_LogBib_NoActiveCheckpoint(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 10, RaceID: 5, BibNumber: 42}},
	}
	// session has no checkpoint for race 5
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID: &eventIDVal,
	}}

	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, sess)
	_, err := svc.LogBib(context.Background(), portsvc.LogBibInput{BibNumber: 42})

	assert.ErrorContains(t, err, "no active checkpoint")
}

func TestCheckpointLogService_LogStatus_UpdatesDNS(t *testing.T) {
	eventIDVal := 1
	var updatedID int
	var updatedStatus entity.RunnerStatus
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 10, RaceID: 5, BibNumber: 42, Status: entity.StatusActive}},
		updateStatus: func(id int, status entity.RunnerStatus) {
			updatedID = id
			updatedStatus = status
		},
	}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}

	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, sess)
	err := svc.LogStatus(context.Background(), 42, entity.StatusDNS)

	require.NoError(t, err)
	assert.Equal(t, 10, updatedID)
	assert.Equal(t, entity.StatusDNS, updatedStatus)
}

func TestCheckpointLogService_LogStatus_NoSession(t *testing.T) {
	svc := newCheckpointLogSvc(
		&mockRunnerRepository{},
		&mockCheckpointLogRepository{},
		&mockActiveSessionRepository{},
	)

	err := svc.LogStatus(context.Background(), 42, entity.StatusDNS)

	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestCheckpointLogService_LogBib_SessionError(t *testing.T) {
	sess := &mockActiveSessionRepository{getErr: errors.New("db down")}
	svc := newCheckpointLogSvc(&mockRunnerRepository{}, &mockCheckpointLogRepository{}, sess)

	_, err := svc.LogBib(context.Background(), portsvc.LogBibInput{BibNumber: 42})

	assert.ErrorContains(t, err, "getting session")
}

func TestCheckpointLogService_LogBib_ExistsError(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 10, RaceID: 5, BibNumber: 42}}}
	logs := &mockCheckpointLogRepository{existsErr: errors.New("db down")}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 5, CheckpointID: 99}},
	}}

	svc := newCheckpointLogSvc(runners, logs, sess)
	_, err := svc.LogBib(context.Background(), portsvc.LogBibInput{BibNumber: 42})

	assert.ErrorContains(t, err, "checking duplicate")
}

func TestCheckpointLogService_LogBib_CreateError(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 10, RaceID: 5, BibNumber: 42}}}
	logs := &mockCheckpointLogRepository{createErr: errors.New("db down")}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 5, CheckpointID: 99}},
	}}

	svc := newCheckpointLogSvc(runners, logs, sess)
	_, err := svc.LogBib(context.Background(), portsvc.LogBibInput{BibNumber: 42})

	assert.ErrorContains(t, err, "creating log")
}

func TestCheckpointLogService_LogStatus_SessionError(t *testing.T) {
	sess := &mockActiveSessionRepository{getErr: errors.New("db down")}
	svc := newCheckpointLogSvc(&mockRunnerRepository{}, &mockCheckpointLogRepository{}, sess)

	err := svc.LogStatus(context.Background(), 42, entity.StatusDNS)

	assert.ErrorContains(t, err, "getting session")
}

func TestCheckpointLogService_LogStatus_RunnerError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	svc := newCheckpointLogSvc(&mockRunnerRepository{}, &mockCheckpointLogRepository{}, sess)

	err := svc.LogStatus(context.Background(), 999, entity.StatusDNS)

	assert.ErrorContains(t, err, "999")
}

// --- CheckpointLogService.ListByRace ---

func TestCheckpointLogService_ListByRace(t *testing.T) {
	logs := &mockCheckpointLogRepository{
		logs: []entity.CheckpointLog{
			{ID: 1, RunnerID: 10, CheckpointID: 5},
			{ID: 2, RunnerID: 11, CheckpointID: 5},
		},
	}
	svc := newCheckpointLogSvc(&mockRunnerRepository{}, logs, &mockActiveSessionRepository{})

	result, err := svc.ListByRace(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}
