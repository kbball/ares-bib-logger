package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/repository"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
)

func TestRaceRepo_CreateAndGet(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	repo := repository.NewRaceRepo(db)
	ctx := context.Background()

	race, err := repo.Create(ctx, eventID, "GDR 100")
	require.NoError(t, err)
	assert.Greater(t, race.ID, 0)
	assert.Equal(t, eventID, race.EventID)
	assert.Equal(t, "GDR 100", race.Name)
	assert.False(t, race.RosterLocked)
	assert.False(t, race.OrderLocked)

	fetched, err := repo.Get(ctx, race.ID)
	require.NoError(t, err)
	assert.Equal(t, race.ID, fetched.ID)
}

func TestRaceRepo_List(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	repo := repository.NewRaceRepo(db)
	ctx := context.Background()

	_, err := repo.Create(ctx, eventID, "Race 1")
	require.NoError(t, err)
	_, err = repo.Create(ctx, eventID, "Race 2")
	require.NoError(t, err)

	races, err := repo.List(ctx, eventID)
	require.NoError(t, err)
	assert.Len(t, races, 2)
}

func TestRaceRepo_LockRoster(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	repo := repository.NewRaceRepo(db)
	ctx := context.Background()

	race, err := repo.Create(ctx, eventID, "GDR 100")
	require.NoError(t, err)
	assert.False(t, race.RosterLocked)

	err = repo.LockRoster(ctx, race.ID)
	require.NoError(t, err)

	fetched, err := repo.Get(ctx, race.ID)
	require.NoError(t, err)
	assert.True(t, fetched.RosterLocked)
}

func TestRaceRepo_LockOrder(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	repo := repository.NewRaceRepo(db)
	ctx := context.Background()

	race, err := repo.Create(ctx, eventID, "GDR 100")
	require.NoError(t, err)

	err = repo.LockOrder(ctx, race.ID)
	require.NoError(t, err)

	fetched, err := repo.Get(ctx, race.ID)
	require.NoError(t, err)
	assert.True(t, fetched.OrderLocked)
}

func TestRaceRepo_Delete(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	repo := repository.NewRaceRepo(db)
	ctx := context.Background()

	race, err := repo.Create(ctx, eventID, "To Delete")
	require.NoError(t, err)

	err = repo.Delete(ctx, race.ID)
	require.NoError(t, err)

	_, err = repo.Get(ctx, race.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
