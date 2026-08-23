package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

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
	checkpoints ...*mockCheckpointRepository,
) *service.CheckpointLogService {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}
	if len(checkpoints) > 0 {
		cps = checkpoints[0]
	}
	races := &mockRaceRepository{races: map[int]entity.Race{}}
	return service.NewCheckpointLogService(runners, cps, logs, sess, races, time.UTC)
}

// newCheckpointLogSvcWithRaces is like newCheckpointLogSvc but also takes a
// race repo, for tests of station-wide commands (checkpoint/count/search/dup)
// that need real race data.
func newCheckpointLogSvcWithRaces(
	runners *mockRunnerRepository,
	checkpoints *mockCheckpointRepository,
	logs *mockCheckpointLogRepository,
	sess *mockActiveSessionRepository,
	races *mockRaceRepository,
) *service.CheckpointLogService {
	return service.NewCheckpointLogService(runners, checkpoints, logs, sess, races, time.UTC)
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

// --- CheckpointLogService.CorrectLog ---

func TestCheckpointLogService_CorrectLog_CreatesLog(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, Status: entity.StatusUnknown}},
	}
	logs := &mockCheckpointLogRepository{}
	svc := newCheckpointLogSvc(runners, logs, &mockActiveSessionRepository{})

	log, err := svc.CorrectLog(context.Background(), 1, 5, 100, "", "14:32")
	require.NoError(t, err)
	assert.Equal(t, 1, log.RunnerID)
	assert.Equal(t, 5, log.CheckpointID)
	assert.Equal(t, entity.SourceCorrection, log.Source)
	assert.Equal(t, entity.StatusActive, runners.runners[0].Status)
}

func TestCheckpointLogService_CorrectLog_OverwritesExistingLog(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, Status: entity.StatusActive}},
	}
	logs := &mockCheckpointLogRepository{
		logs: []entity.CheckpointLog{{ID: 9, RunnerID: 1, CheckpointID: 5, Source: entity.SourceMeshtastic}},
	}
	svc := newCheckpointLogSvc(runners, logs, &mockActiveSessionRepository{})

	log, err := svc.CorrectLog(context.Background(), 1, 5, 100, "", "14:32")
	require.NoError(t, err)
	assert.Equal(t, 9, log.ID)
	assert.Equal(t, entity.SourceCorrection, log.Source)
	require.Len(t, logs.logs, 1)
}

func TestCheckpointLogService_CorrectLog_ExplicitDate(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, Status: entity.StatusUnknown}},
	}
	logs := &mockCheckpointLogRepository{}
	svc := newCheckpointLogSvc(runners, logs, &mockActiveSessionRepository{})

	log, err := svc.CorrectLog(context.Background(), 1, 5, 100, "2026-08-22", "23:58")
	require.NoError(t, err)
	assert.Equal(t, 2026, log.RecordedAt.Year())
	assert.Equal(t, time.Month(8), log.RecordedAt.Month())
	assert.Equal(t, 22, log.RecordedAt.Day())
	assert.Equal(t, 23, log.RecordedAt.Hour())
	assert.Equal(t, 58, log.RecordedAt.Minute())
}

func TestCheckpointLogService_CorrectLog_InvalidDate(t *testing.T) {
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100}}}
	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	_, err := svc.CorrectLog(context.Background(), 1, 5, 100, "not-a-date", "14:32")
	assert.ErrorContains(t, err, "cannot parse date")
}

func TestCheckpointLogService_CorrectLog_InvalidTime(t *testing.T) {
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100}}}
	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	_, err := svc.CorrectLog(context.Background(), 1, 5, 100, "", "not-a-time")
	assert.ErrorContains(t, err, "cannot parse time")
}

func TestCheckpointLogService_CorrectLog_BibNotFound(t *testing.T) {
	runners := &mockRunnerRepository{}
	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	_, err := svc.CorrectLog(context.Background(), 1, 5, 999, "", "14:32")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCheckpointLogService_CorrectLog_ListRunnersError(t *testing.T) {
	runners := &mockRunnerRepository{listErr: errors.New("db down")}
	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	_, err := svc.CorrectLog(context.Background(), 1, 5, 100, "", "14:32")
	assert.ErrorContains(t, err, "db down")
}

func TestCheckpointLogService_CorrectLog_UpsertError(t *testing.T) {
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100}}}
	logs := &mockCheckpointLogRepository{upsertErr: errors.New("upsert failed")}
	svc := newCheckpointLogSvc(runners, logs, &mockActiveSessionRepository{})

	_, err := svc.CorrectLog(context.Background(), 1, 5, 100, "", "14:32")
	assert.ErrorContains(t, err, "upsert failed")
}

func TestCheckpointLogService_CorrectLog_UpdateStatusError(t *testing.T) {
	runners := &mockRunnerRepository{
		runners:         []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, Status: entity.StatusUnknown}},
		updateStatusErr: errors.New("status update failed"),
	}
	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	_, err := svc.CorrectLog(context.Background(), 1, 5, 100, "", "14:32")
	assert.ErrorContains(t, err, "status update failed")
}

// --- CheckpointLogService.DeleteLog ---

func TestCheckpointLogService_DeleteLog_Success(t *testing.T) {
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100}}}
	logs := &mockCheckpointLogRepository{
		logs: []entity.CheckpointLog{{ID: 9, RunnerID: 1, CheckpointID: 5}},
	}
	svc := newCheckpointLogSvc(runners, logs, &mockActiveSessionRepository{})

	err := svc.DeleteLog(context.Background(), 1, 5, 100)
	require.NoError(t, err)
	assert.Empty(t, logs.logs)
}

func TestCheckpointLogService_DeleteLog_BibNotFound(t *testing.T) {
	runners := &mockRunnerRepository{}
	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	err := svc.DeleteLog(context.Background(), 1, 5, 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCheckpointLogService_DeleteLog_NoLogAtCheckpoint(t *testing.T) {
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100}}}
	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	err := svc.DeleteLog(context.Background(), 1, 5, 100)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- CheckpointLogService.QueryRunner ---

func dist(v float64) *float64 { return &v }

func TestCheckpointLogService_QueryRunner_NotFound(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	svc := newCheckpointLogSvc(&mockRunnerRepository{}, &mockCheckpointLogRepository{}, sess)

	reply, err := svc.QueryRunner(context.Background(), 999)
	require.NoError(t, err)
	assert.Equal(t, "999 not found", reply)
}

func TestCheckpointLogService_QueryRunner_NoSession(t *testing.T) {
	sess := &mockActiveSessionRepository{}
	svc := newCheckpointLogSvc(&mockRunnerRepository{}, &mockCheckpointLogRepository{}, sess)

	_, err := svc.QueryRunner(context.Background(), 100)
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestCheckpointLogService_QueryRunner_SessionError(t *testing.T) {
	sess := &mockActiveSessionRepository{getErr: errors.New("db down")}
	svc := newCheckpointLogSvc(&mockRunnerRepository{}, &mockCheckpointLogRepository{}, sess)

	_, err := svc.QueryRunner(context.Background(), 100)
	assert.ErrorContains(t, err, "getting session")
}

func TestCheckpointLogService_QueryRunner_NeverSeen(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 5, BibNumber: 100, FirstName: "Jane", LastName: "Doe", Status: entity.StatusActive}},
	}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 5, DisplayOrder: 1, DisplayName: "AS1", DistanceFromStart: dist(5.0)},
	}}
	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, sess, cps)

	reply, err := svc.QueryRunner(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, "100 Jane Doe: ACTIVE not yet seen", reply)
}

func TestCheckpointLogService_QueryRunner_LastStationNoPace(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 5, BibNumber: 100, FirstName: "Jane", LastName: "Doe", Status: entity.StatusDNF}},
	}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 5, DisplayOrder: 1, DisplayName: "AS1", DistanceFromStart: dist(5.0)},
	}}
	logs := &mockCheckpointLogRepository{logs: []entity.CheckpointLog{
		{RunnerID: 1, CheckpointID: 1, RecordedAt: mustParseTime("2026-06-14T14:32:00Z")},
	}}
	svc := newCheckpointLogSvc(runners, logs, sess, cps)

	reply, err := svc.QueryRunner(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, "100 Jane Doe: DNF last AS1 14:32", reply)
}

func TestCheckpointLogService_QueryRunner_LastStationWithPace(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 5, BibNumber: 100, FirstName: "Jane", LastName: "Doe", Status: entity.StatusActive}},
	}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 5, DisplayOrder: 1, DisplayName: "AS1", DistanceFromStart: dist(10.0)},
		2: {ID: 2, RaceID: 5, DisplayOrder: 2, DisplayName: "AS2", DistanceFromStart: dist(20.0)},
	}}
	logs := &mockCheckpointLogRepository{logs: []entity.CheckpointLog{
		{RunnerID: 1, CheckpointID: 1, RecordedAt: mustParseTime("2026-06-14T10:00:00Z")},
		{RunnerID: 1, CheckpointID: 2, RecordedAt: mustParseTime("2026-06-14T11:00:00Z")},
	}}
	svc := newCheckpointLogSvc(runners, logs, sess, cps)

	reply, err := svc.QueryRunner(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, "100 Jane Doe: ACTIVE last AS2 11:00 pace 6:00 /mi", reply)
}

func TestCheckpointLogService_QueryRunner_ListCheckpointsError(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 5, BibNumber: 100}},
	}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	cps := &mockCheckpointRepository{listErr: errors.New("db down")}
	svc := newCheckpointLogSvc(runners, &mockCheckpointLogRepository{}, sess, cps)

	_, err := svc.QueryRunner(context.Background(), 100)
	assert.ErrorContains(t, err, "listing checkpoints")
}

func TestCheckpointLogService_QueryRunner_ListLogsError(t *testing.T) {
	eventIDVal := 1
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 5, BibNumber: 100}},
	}
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}
	logs := &mockCheckpointLogRepository{listErr: errors.New("db down")}
	svc := newCheckpointLogSvc(runners, logs, sess, cps)

	_, err := svc.QueryRunner(context.Background(), 100)
	assert.ErrorContains(t, err, "listing checkpoint logs")
}

// --- CheckpointLogService.StationCheckpoints ---

func TestCheckpointLogService_StationCheckpoints_SingleRace(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 10}},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1, Name: "GDR"}}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		10: {ID: 10, RaceID: 1, Code: "AS6", DisplayName: "Aid Station 6"},
	}}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, cps, &mockCheckpointLogRepository{}, sess, races)

	reply, err := svc.StationCheckpoints(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "GDR AS6", reply)
}

func TestCheckpointLogService_StationCheckpoints_MultiRace(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID: &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{
			{RaceID: 1, CheckpointID: 10},
			{RaceID: 2, CheckpointID: 20},
		},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{
		1: {ID: 1, EventID: 1, Name: "100M"},
		2: {ID: 2, EventID: 1, Name: "50M"},
	}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		10: {ID: 10, RaceID: 1, Code: "StoverRoadOut"},
		20: {ID: 20, RaceID: 2, Code: "PocketDepart"},
	}}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, cps, &mockCheckpointLogRepository{}, sess, races)

	reply, err := svc.StationCheckpoints(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "100M StoverRoadOut | 50M PocketDepart", reply)
}

func TestCheckpointLogService_StationCheckpoints_NoSession(t *testing.T) {
	sess := &mockActiveSessionRepository{}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	_, err := svc.StationCheckpoints(context.Background())
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestCheckpointLogService_StationCheckpoints_SessionError(t *testing.T) {
	sess := &mockActiveSessionRepository{getErr: errors.New("db down")}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	_, err := svc.StationCheckpoints(context.Background())
	assert.ErrorContains(t, err, "getting session")
}

func TestCheckpointLogService_StationCheckpoints_NoActiveCheckpoints(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	reply, err := svc.StationCheckpoints(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "no active checkpoint set for any race", reply)
}

func TestCheckpointLogService_StationCheckpoints_RaceError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 10}},
	}}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	_, err := svc.StationCheckpoints(context.Background())
	assert.ErrorContains(t, err, "getting race")
}

func TestCheckpointLogService_StationCheckpoints_CheckpointError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 10}},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1, Name: "GDR"}}}
	cps := &mockCheckpointRepository{getErr: errors.New("db down")}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, cps, &mockCheckpointLogRepository{}, sess, races)

	_, err := svc.StationCheckpoints(context.Background())
	assert.ErrorContains(t, err, "getting checkpoint")
}

// --- CheckpointLogService.StationCount ---

func TestCheckpointLogService_StationCount_SingleRace(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 10}},
	}}
	logs := &mockCheckpointLogRepository{logs: []entity.CheckpointLog{
		{RunnerID: 1, CheckpointID: 10},
		{RunnerID: 2, CheckpointID: 10},
	}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		10: {ID: 10, RaceID: 1, Code: "AS6"},
	}}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, cps, logs, sess, &mockRaceRepository{})

	reply, err := svc.StationCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "logged 2 at AS6", reply)
}

func TestCheckpointLogService_StationCount_MultiRace(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID: &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{
			{RaceID: 1, CheckpointID: 10},
			{RaceID: 2, CheckpointID: 20},
		},
	}}
	logs := &mockCheckpointLogRepository{logs: []entity.CheckpointLog{
		{RunnerID: 1, CheckpointID: 10},
		{RunnerID: 2, CheckpointID: 20},
		{RunnerID: 3, CheckpointID: 20},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{
		1: {ID: 1, EventID: 1, Name: "100M"},
		2: {ID: 2, EventID: 1, Name: "50M"},
	}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		10: {ID: 10, RaceID: 1, Code: "AS1"},
		20: {ID: 20, RaceID: 2, Code: "AS2"},
	}}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, cps, logs, sess, races)

	reply, err := svc.StationCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "logged 3 total (100M AS1=1, 50M AS2=2)", reply)
}

func TestCheckpointLogService_StationCount_NoSession(t *testing.T) {
	sess := &mockActiveSessionRepository{}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	_, err := svc.StationCount(context.Background())
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestCheckpointLogService_StationCount_NoActiveCheckpoints(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	reply, err := svc.StationCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "no active checkpoint set for any race", reply)
}

func TestCheckpointLogService_StationCount_ListLogsError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 10}},
	}}
	logs := &mockCheckpointLogRepository{listErr: errors.New("db down")}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, logs, sess, &mockRaceRepository{})

	_, err := svc.StationCount(context.Background())
	assert.ErrorContains(t, err, "listing logs")
}

// --- CheckpointLogService.SearchRunners ---

func TestCheckpointLogService_SearchRunners_NoMatches(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1, Name: "GDR"}}}
	runners := &mockRunnerRepository{runners: []entity.Runner{
		{ID: 1, RaceID: 1, BibNumber: 100, FirstName: "Alice", LastName: "Smith"},
	}}
	svc := newCheckpointLogSvcWithRaces(runners, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, races)

	reply, err := svc.SearchRunners(context.Background(), "jones")
	require.NoError(t, err)
	assert.Equal(t, `no runners matching "jones"`, reply)
}

func TestCheckpointLogService_SearchRunners_CaseInsensitiveMatch(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1, Name: "GDR"}}}
	runners := &mockRunnerRepository{runners: []entity.Runner{
		{ID: 1, RaceID: 1, BibNumber: 100, FirstName: "Alice", LastName: "Smith"},
		{ID: 2, RaceID: 1, BibNumber: 101, FirstName: "Bob", LastName: "Jones"},
	}}
	svc := newCheckpointLogSvcWithRaces(runners, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, races)

	reply, err := svc.SearchRunners(context.Background(), "SMITH")
	require.NoError(t, err)
	assert.Equal(t, "1 match(es): 100 Alice Smith (GDR)", reply)
}

func TestCheckpointLogService_SearchRunners_TruncatesLongResults(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1, Name: "GDR"}}}
	var runnerList []entity.Runner
	for i := 0; i < 6; i++ {
		runnerList = append(runnerList, entity.Runner{
			ID: i + 1, RaceID: 1, BibNumber: 100 + i, FirstName: "Runner", LastName: "Smith",
		})
	}
	runners := &mockRunnerRepository{runners: runnerList}
	svc := newCheckpointLogSvcWithRaces(runners, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, races)

	reply, err := svc.SearchRunners(context.Background(), "smith")
	require.NoError(t, err)
	assert.Contains(t, reply, "6 match(es):")
	assert.Contains(t, reply, "+1 more")
}

func TestCheckpointLogService_SearchRunners_NoSession(t *testing.T) {
	sess := &mockActiveSessionRepository{}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	_, err := svc.SearchRunners(context.Background(), "smith")
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestCheckpointLogService_SearchRunners_ListRunnersError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1, Name: "GDR"}}}
	runners := &mockRunnerRepository{listErr: errors.New("db down")}
	svc := newCheckpointLogSvcWithRaces(runners, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, races)

	_, err := svc.SearchRunners(context.Background(), "smith")
	assert.ErrorContains(t, err, "listing runners")
}

// --- CheckpointLogService.CheckDuplicate ---

func TestCheckpointLogService_CheckDuplicate_NotFound(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	reply, err := svc.CheckDuplicate(context.Background(), 999)
	require.NoError(t, err)
	assert.Equal(t, "999 not found", reply)
}

func TestCheckpointLogService_CheckDuplicate_NoActiveCheckpointForRace(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{EventID: &eventIDVal}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1, Name: "GDR"}}}
	runners := &mockRunnerRepository{runners: []entity.Runner{
		{ID: 1, RaceID: 1, BibNumber: 100, FirstName: "Alice", LastName: "Smith"},
	}}
	svc := newCheckpointLogSvcWithRaces(runners, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, races)

	reply, err := svc.CheckDuplicate(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, "100 Alice Smith: no active checkpoint for GDR", reply)
}

func TestCheckpointLogService_CheckDuplicate_AlreadyLogged(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 10}},
	}}
	runners := &mockRunnerRepository{runners: []entity.Runner{
		{ID: 1, RaceID: 1, BibNumber: 100, FirstName: "Alice", LastName: "Smith"},
	}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		10: {ID: 10, RaceID: 1, Code: "AS6"},
	}}
	logs := &mockCheckpointLogRepository{logs: []entity.CheckpointLog{
		{RunnerID: 1, CheckpointID: 10, RecordedAt: mustParseTime("2026-06-14T14:32:00Z")},
	}}
	svc := newCheckpointLogSvcWithRaces(runners, cps, logs, sess, &mockRaceRepository{})

	reply, err := svc.CheckDuplicate(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, "100 Alice Smith: already logged at AS6 14:32", reply)
}

func TestCheckpointLogService_CheckDuplicate_NotYetLogged(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 10}},
	}}
	runners := &mockRunnerRepository{runners: []entity.Runner{
		{ID: 1, RaceID: 1, BibNumber: 100, FirstName: "Alice", LastName: "Smith"},
	}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		10: {ID: 10, RaceID: 1, Code: "AS6"},
	}}
	svc := newCheckpointLogSvcWithRaces(runners, cps, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	reply, err := svc.CheckDuplicate(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, "100 Alice Smith: not yet logged at AS6", reply)
}

func TestCheckpointLogService_CheckDuplicate_NoSession(t *testing.T) {
	sess := &mockActiveSessionRepository{}
	svc := newCheckpointLogSvcWithRaces(&mockRunnerRepository{}, &mockCheckpointRepository{}, &mockCheckpointLogRepository{}, sess, &mockRaceRepository{})

	_, err := svc.CheckDuplicate(context.Background(), 100)
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
