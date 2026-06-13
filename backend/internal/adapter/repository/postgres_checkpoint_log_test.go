package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/repository"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

func seedRunnerAndCheckpoint(t *testing.T) (runnerID, checkpointID int) {
	t.Helper()
	db := integrationDB
	ctx := context.Background()

	eventID := seedEvent(t, db)
	raceID := seedRace(t, db, eventID)

	err := repository.NewRunnerRepo(db).BulkCreate(ctx, []entity.Runner{
		{RaceID: raceID, BibNumber: 42, FirstName: "Test", LastName: "Runner", SortOrder: 1, Status: entity.StatusActive},
	})
	require.NoError(t, err)

	runners, err := repository.NewRunnerRepo(db).List(ctx, raceID)
	require.NoError(t, err)
	require.Len(t, runners, 1)

	cp, err := repository.NewCheckpointRepo(db).Create(ctx, entity.Checkpoint{
		RaceID: raceID, Code: "AS1", DisplayName: "Aid 1", DisplayOrder: 1,
	})
	require.NoError(t, err)

	return runners[0].ID, cp.ID
}

func TestCheckpointLogRepo_CreateAndExists(t *testing.T) {
	requireDB(t)
	runnerID, checkpointID := seedRunnerAndCheckpoint(t)
	repo := repository.NewCheckpointLogRepo(integrationDB)
	ctx := context.Background()

	exists, err := repo.ExistsByRunnerAndCheckpoint(ctx, runnerID, checkpointID)
	require.NoError(t, err)
	assert.False(t, exists)

	log, err := repo.Create(ctx, entity.CheckpointLog{
		RunnerID:     runnerID,
		CheckpointID: checkpointID,
		RecordedAt:   time.Now().UTC(),
		Source:       entity.SourceManual,
		RawMessage:   "test",
	})
	require.NoError(t, err)
	assert.Greater(t, log.ID, 0)
	assert.Equal(t, runnerID, log.RunnerID)

	exists, err = repo.ExistsByRunnerAndCheckpoint(ctx, runnerID, checkpointID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCheckpointLogRepo_ListByRaceAndCheckpoint(t *testing.T) {
	requireDB(t)
	runnerID, checkpointID := seedRunnerAndCheckpoint(t)
	repo := repository.NewCheckpointLogRepo(integrationDB)
	ctx := context.Background()

	// Fetch raceID from runner
	runner, err := repository.NewRunnerRepo(integrationDB).Get(ctx, runnerID)
	require.NoError(t, err)

	_, err = repo.Create(ctx, entity.CheckpointLog{
		RunnerID: runnerID, CheckpointID: checkpointID,
		RecordedAt: time.Now().UTC(), Source: entity.SourceManual,
	})
	require.NoError(t, err)

	logs, err := repo.ListByRaceAndCheckpoint(ctx, runner.RaceID, checkpointID)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, runnerID, logs[0].RunnerID)
}

func TestCheckpointLogRepo_ListByRace(t *testing.T) {
	requireDB(t)
	runnerID, checkpointID := seedRunnerAndCheckpoint(t)
	repo := repository.NewCheckpointLogRepo(integrationDB)
	ctx := context.Background()

	runner, err := repository.NewRunnerRepo(integrationDB).Get(ctx, runnerID)
	require.NoError(t, err)

	_, err = repo.Create(ctx, entity.CheckpointLog{
		RunnerID: runnerID, CheckpointID: checkpointID,
		RecordedAt: time.Now().UTC(), Source: entity.SourceMeshtastic, RawMessage: "raw",
	})
	require.NoError(t, err)

	logs, err := repo.ListByRace(ctx, runner.RaceID)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}
