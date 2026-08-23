package pace_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/pace"
)

func makeRunner(status entity.RunnerStatus) entity.Runner {
	return entity.Runner{ID: 1, RaceID: 1, BibNumber: 100, Status: status}
}

func makeCheckpoint(id, order int, dist *float64) entity.Checkpoint {
	return entity.Checkpoint{ID: id, RaceID: 1, DisplayOrder: order, DistanceFromStart: dist}
}

func makeLog(runnerID, checkpointID int, recordedAt time.Time) entity.CheckpointLog {
	return entity.CheckpointLog{RunnerID: runnerID, CheckpointID: checkpointID, RecordedAt: recordedAt}
}

func dist(v float64) *float64 { return &v }

func mustParse(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestComputeRunnerPace_TerminalStatusesReturnEmpty(t *testing.T) {
	for _, status := range []entity.RunnerStatus{
		entity.StatusDNS, entity.StatusDNF, entity.StatusMoved, entity.StatusFinished,
	} {
		result := pace.ComputeRunnerPace(makeRunner(status), nil, nil)
		assert.Nil(t, result.PaceMinPerMile, "status %s", status)
		assert.Nil(t, result.LastLoggedDist, "status %s", status)
		assert.Nil(t, result.LastLoggedAt, "status %s", status)
	}
}

func TestComputeRunnerPace_NoLogsReturnsEmpty(t *testing.T) {
	cps := []entity.Checkpoint{makeCheckpoint(1, 1, dist(10.0))}
	result := pace.ComputeRunnerPace(makeRunner(entity.StatusActive), cps, nil)
	assert.Nil(t, result.PaceMinPerMile)
	assert.Nil(t, result.LastLoggedDist)
}

func TestComputeRunnerPace_OneDistanceCheckpoint_NoPaceButLastPosition(t *testing.T) {
	cps := []entity.Checkpoint{makeCheckpoint(1, 1, dist(10.0)), makeCheckpoint(2, 2, dist(20.0))}
	at := mustParse("2026-06-14T10:00:00Z")
	logs := []entity.CheckpointLog{makeLog(1, 1, at)}

	result := pace.ComputeRunnerPace(makeRunner(entity.StatusActive), cps, logs)
	assert.Nil(t, result.PaceMinPerMile)
	require.NotNil(t, result.LastLoggedDist)
	assert.Equal(t, 10.0, *result.LastLoggedDist)
	require.NotNil(t, result.LastLoggedAt)
	assert.True(t, at.Equal(*result.LastLoggedAt))
}

func TestComputeRunnerPace_TwoDistanceCheckpoints(t *testing.T) {
	cps := []entity.Checkpoint{makeCheckpoint(1, 1, dist(10.0)), makeCheckpoint(2, 2, dist(20.0))}
	// 60 minutes between checkpoints, 10 miles apart -> 6 min/mi
	logs := []entity.CheckpointLog{
		makeLog(1, 1, mustParse("2026-06-14T10:00:00Z")),
		makeLog(1, 2, mustParse("2026-06-14T11:00:00Z")),
	}

	result := pace.ComputeRunnerPace(makeRunner(entity.StatusActive), cps, logs)
	require.NotNil(t, result.PaceMinPerMile)
	assert.InDelta(t, 6.0, *result.PaceMinPerMile, 0.001)
	assert.Equal(t, 20.0, *result.LastLoggedDist)
}

func TestComputeRunnerPace_UsesLastTwoWhenMoreThanTwoLogged(t *testing.T) {
	cps := []entity.Checkpoint{
		makeCheckpoint(1, 1, dist(5.0)),
		makeCheckpoint(2, 2, dist(15.0)),
		makeCheckpoint(3, 3, dist(25.0)),
	}
	// Between cp2 and cp3: 10 miles in 90 min -> 9 min/mi
	logs := []entity.CheckpointLog{
		makeLog(1, 1, mustParse("2026-06-14T09:00:00Z")),
		makeLog(1, 2, mustParse("2026-06-14T10:00:00Z")),
		makeLog(1, 3, mustParse("2026-06-14T11:30:00Z")),
	}

	result := pace.ComputeRunnerPace(makeRunner(entity.StatusActive), cps, logs)
	require.NotNil(t, result.PaceMinPerMile)
	assert.InDelta(t, 9.0, *result.PaceMinPerMile, 0.001)
}

func TestComputeRunnerPace_SkipsCheckpointsWithNilDistance(t *testing.T) {
	cps := []entity.Checkpoint{
		makeCheckpoint(1, 1, dist(10.0)),
		makeCheckpoint(2, 2, nil),
		makeCheckpoint(3, 3, dist(20.0)),
	}
	logs := []entity.CheckpointLog{
		makeLog(1, 1, mustParse("2026-06-14T10:00:00Z")),
		makeLog(1, 2, mustParse("2026-06-14T10:30:00Z")),
		makeLog(1, 3, mustParse("2026-06-14T11:00:00Z")),
	}

	result := pace.ComputeRunnerPace(makeRunner(entity.StatusActive), cps, logs)
	require.NotNil(t, result.PaceMinPerMile)
	assert.InDelta(t, 6.0, *result.PaceMinPerMile, 0.001)
}

func TestComputeRunnerPace_ZeroDistanceDeltaReturnsNilPace(t *testing.T) {
	cps := []entity.Checkpoint{makeCheckpoint(1, 1, dist(10.0)), makeCheckpoint(2, 2, dist(10.0))}
	logs := []entity.CheckpointLog{
		makeLog(1, 1, mustParse("2026-06-14T10:00:00Z")),
		makeLog(1, 2, mustParse("2026-06-14T11:00:00Z")),
	}

	result := pace.ComputeRunnerPace(makeRunner(entity.StatusActive), cps, logs)
	assert.Nil(t, result.PaceMinPerMile)
}

func TestComputeRunnerPace_IgnoresOtherRunnersLogs(t *testing.T) {
	cps := []entity.Checkpoint{makeCheckpoint(1, 1, dist(10.0)), makeCheckpoint(2, 2, dist(20.0))}
	logs := []entity.CheckpointLog{
		makeLog(99, 1, mustParse("2026-06-14T10:00:00Z")),
		makeLog(99, 2, mustParse("2026-06-14T11:00:00Z")),
	}

	result := pace.ComputeRunnerPace(makeRunner(entity.StatusActive), cps, logs)
	assert.Nil(t, result.PaceMinPerMile)
}

func TestProjectArrival(t *testing.T) {
	p6 := 6.0
	d10 := 10.0
	at := mustParse("2026-06-14T10:00:00Z")
	basePace := pace.RunnerPace{PaceMinPerMile: &p6, LastLoggedDist: &d10, LastLoggedAt: &at}

	t.Run("nil pace", func(t *testing.T) {
		p := basePace
		p.PaceMinPerMile = nil
		assert.Nil(t, pace.ProjectArrival(p, 20))
	})

	t.Run("nil last dist", func(t *testing.T) {
		p := basePace
		p.LastLoggedDist = nil
		assert.Nil(t, pace.ProjectArrival(p, 20))
	})

	t.Run("nil last at", func(t *testing.T) {
		p := basePace
		p.LastLoggedAt = nil
		assert.Nil(t, pace.ProjectArrival(p, 20))
	})

	t.Run("target not ahead of last position", func(t *testing.T) {
		assert.Nil(t, pace.ProjectArrival(basePace, 10))
	})

	t.Run("projects arrival", func(t *testing.T) {
		got := pace.ProjectArrival(basePace, 20)
		require.NotNil(t, got)
		want := mustParse("2026-06-14T11:00:00Z")
		assert.True(t, want.Equal(*got), "want %v got %v", want, *got)
	})
}

func TestFormatPace(t *testing.T) {
	assert.Equal(t, "6:00 /mi", pace.FormatPace(6.0))
	assert.Equal(t, "12:30 /mi", pace.FormatPace(12.5))
}

func TestLastLoggedCheckpoint_NoLogsReturnsFalse(t *testing.T) {
	_, _, ok := pace.LastLoggedCheckpoint(nil, nil, 1)
	assert.False(t, ok)
}

func TestLastLoggedCheckpoint_ReturnsHighestDisplayOrderRegardlessOfStatus(t *testing.T) {
	cps := []entity.Checkpoint{
		makeCheckpoint(1, 1, dist(5.0)),
		makeCheckpoint(2, 2, dist(15.0)),
	}
	at := mustParse("2026-06-14T10:00:00Z")
	logs := []entity.CheckpointLog{
		makeLog(1, 1, at),
		makeLog(1, 2, at.Add(30*time.Minute)),
	}

	cp, log, ok := pace.LastLoggedCheckpoint(cps, logs, 1)
	require.True(t, ok)
	assert.Equal(t, 2, cp.ID)
	assert.Equal(t, 2, log.CheckpointID)
}

func TestLastLoggedCheckpoint_AvailableForTerminalStatuses(t *testing.T) {
	// Even though ComputeRunnerPace returns empty for DNF runners,
	// LastLoggedCheckpoint must still report their last known station.
	cps := []entity.Checkpoint{makeCheckpoint(1, 1, dist(5.0))}
	logs := []entity.CheckpointLog{makeLog(1, 1, mustParse("2026-06-14T10:00:00Z"))}

	dnfResult := pace.ComputeRunnerPace(makeRunner(entity.StatusDNF), cps, logs)
	assert.Nil(t, dnfResult.LastLoggedDist, "ComputeRunnerPace should stay empty for DNF")

	cp, _, ok := pace.LastLoggedCheckpoint(cps, logs, 1)
	require.True(t, ok)
	assert.Equal(t, 1, cp.ID)
}
