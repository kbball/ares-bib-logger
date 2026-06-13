package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/repository"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

func makeRunners(raceID int) []entity.Runner {
	return []entity.Runner{
		{RaceID: raceID, BibNumber: 100, FirstName: "Alice", LastName: "Smith", SortOrder: 1, Status: entity.StatusUnknown},
		{RaceID: raceID, BibNumber: 101, FirstName: "Bob", LastName: "Jones", SortOrder: 2, Status: entity.StatusUnknown},
		{RaceID: raceID, BibNumber: 102, FirstName: "Carol", LastName: "Lee", SortOrder: 3, Status: entity.StatusUnknown},
	}
}

func TestRunnerRepo_BulkCreateAndList(t *testing.T) {
	db := requireDB(t)
	raceID := seedRace(t, db, seedEvent(t, db))
	repo := repository.NewRunnerRepo(db)
	ctx := context.Background()

	err := repo.BulkCreate(ctx, makeRunners(raceID))
	require.NoError(t, err)

	runners, err := repo.List(ctx, raceID)
	require.NoError(t, err)
	require.Len(t, runners, 3)
	assert.Equal(t, 100, runners[0].BibNumber)
	assert.Equal(t, "Alice", runners[0].FirstName)
	assert.Equal(t, "Smith", runners[0].LastName)
	assert.Equal(t, 1, runners[0].SortOrder)
}

func TestRunnerRepo_BulkCreate_Empty(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewRunnerRepo(db)
	err := repo.BulkCreate(context.Background(), nil)
	assert.NoError(t, err)
}

func TestRunnerRepo_GetByBibInEvent(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	raceID := seedRace(t, db, eventID)
	repo := repository.NewRunnerRepo(db)
	ctx := context.Background()

	err := repo.BulkCreate(ctx, makeRunners(raceID))
	require.NoError(t, err)

	runner, err := repo.GetByBibInEvent(ctx, eventID, 101)
	require.NoError(t, err)
	assert.Equal(t, 101, runner.BibNumber)
	assert.Equal(t, raceID, runner.RaceID)
}

func TestRunnerRepo_GetByBibInEvent_NotFound(t *testing.T) {
	db := requireDB(t)
	eventID := seedEvent(t, db)
	repo := repository.NewRunnerRepo(db)

	_, err := repo.GetByBibInEvent(context.Background(), eventID, 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRunnerRepo_UpdateStatus(t *testing.T) {
	db := requireDB(t)
	raceID := seedRace(t, db, seedEvent(t, db))
	repo := repository.NewRunnerRepo(db)
	ctx := context.Background()

	err := repo.BulkCreate(ctx, makeRunners(raceID))
	require.NoError(t, err)

	runners, err := repo.List(ctx, raceID)
	require.NoError(t, err)

	err = repo.UpdateStatus(ctx, runners[0].ID, entity.StatusDNS)
	require.NoError(t, err)

	updated, err := repo.Get(ctx, runners[0].ID)
	require.NoError(t, err)
	assert.Equal(t, entity.StatusDNS, updated.Status)
}

func TestRunnerRepo_MaxSortOrder(t *testing.T) {
	db := requireDB(t)
	raceID := seedRace(t, db, seedEvent(t, db))
	repo := repository.NewRunnerRepo(db)
	ctx := context.Background()

	max, err := repo.MaxSortOrder(ctx, raceID)
	require.NoError(t, err)
	assert.Equal(t, 0, max)

	err = repo.BulkCreate(ctx, makeRunners(raceID))
	require.NoError(t, err)

	max, err = repo.MaxSortOrder(ctx, raceID)
	require.NoError(t, err)
	assert.Equal(t, 3, max)
}
