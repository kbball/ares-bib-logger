package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/repository"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

func TestActiveSessionRepo_GetEmpty(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewActiveSessionRepo(db)

	sess, err := repo.Get(context.Background())
	require.NoError(t, err)
	assert.Nil(t, sess.EventID)
	assert.Empty(t, sess.Checkpoints)
}

func TestActiveSessionRepo_SetEvent(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	repo := repository.NewActiveSessionRepo(db)
	ctx := context.Background()

	err := repo.SetEvent(ctx, eventID)
	require.NoError(t, err)

	sess, err := repo.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, sess.EventID)
	assert.Equal(t, eventID, *sess.EventID)
}

func TestActiveSessionRepo_SetCheckpoint(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	raceID := seedRace(t, db, eventID)
	repo := repository.NewActiveSessionRepo(db)
	ctx := context.Background()

	cp, err := repository.NewCheckpointRepo(db).Create(ctx, entity.Checkpoint{
		RaceID: raceID, Code: "AS1", DisplayName: "Aid 1", DisplayOrder: 1,
	})
	require.NoError(t, err)

	err = repo.SetCheckpoint(ctx, raceID, cp.ID)
	require.NoError(t, err)

	sess, err := repo.Get(ctx)
	require.NoError(t, err)
	require.Len(t, sess.Checkpoints, 1)
	assert.Equal(t, raceID, sess.Checkpoints[0].RaceID)
	assert.Equal(t, cp.ID, sess.Checkpoints[0].CheckpointID)
}

func TestActiveSessionRepo_SetCheckpoint_Upsert(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	raceID := seedRace(t, db, eventID)
	cpRepo := repository.NewCheckpointRepo(db)
	repo := repository.NewActiveSessionRepo(db)
	ctx := context.Background()

	cp1, err := cpRepo.Create(ctx, entity.Checkpoint{RaceID: raceID, Code: "AS1", DisplayName: "Aid 1", DisplayOrder: 1})
	require.NoError(t, err)
	cp2, err := cpRepo.Create(ctx, entity.Checkpoint{RaceID: raceID, Code: "AS2", DisplayName: "Aid 2", DisplayOrder: 2})
	require.NoError(t, err)

	err = repo.SetCheckpoint(ctx, raceID, cp1.ID)
	require.NoError(t, err)
	err = repo.SetCheckpoint(ctx, raceID, cp2.ID) // update same race
	require.NoError(t, err)

	sess, err := repo.Get(ctx)
	require.NoError(t, err)
	require.Len(t, sess.Checkpoints, 1) // still one row for this race
	assert.Equal(t, cp2.ID, sess.Checkpoints[0].CheckpointID)
}

func TestActiveSessionRepo_ClearCheckpoint(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	raceID := seedRace(t, db, eventID)
	repo := repository.NewActiveSessionRepo(db)
	ctx := context.Background()

	cp, err := repository.NewCheckpointRepo(db).Create(ctx, entity.Checkpoint{
		RaceID: raceID, Code: "AS1", DisplayName: "Aid 1", DisplayOrder: 1,
	})
	require.NoError(t, err)

	err = repo.SetCheckpoint(ctx, raceID, cp.ID)
	require.NoError(t, err)

	err = repo.ClearCheckpoint(ctx, raceID)
	require.NoError(t, err)

	sess, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Empty(t, sess.Checkpoints)
}
