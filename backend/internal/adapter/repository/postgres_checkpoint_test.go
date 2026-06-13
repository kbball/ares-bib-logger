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

func TestCheckpointRepo_CreateAndGet(t *testing.T) {
	db := requireDB(t)
	raceID := seedRace(t, db, seedEvent(t, db))
	repo := repository.NewCheckpointRepo(db)
	ctx := context.Background()

	cp, err := repo.Create(ctx, entity.Checkpoint{
		RaceID:       raceID,
		Code:         "AS6",
		DisplayName:  "Aid Station 6",
		DisplayOrder: 1,
	})
	require.NoError(t, err)
	assert.Greater(t, cp.ID, 0)
	assert.Equal(t, "AS6", cp.Code)
	assert.Equal(t, 1, cp.DisplayOrder)

	fetched, err := repo.Get(ctx, cp.ID)
	require.NoError(t, err)
	assert.Equal(t, cp.ID, fetched.ID)
}

func TestCheckpointRepo_List_OrderedByDisplayOrder(t *testing.T) {
	db := requireDB(t)
	raceID := seedRace(t, db, seedEvent(t, db))
	repo := repository.NewCheckpointRepo(db)
	ctx := context.Background()

	_, err := repo.Create(ctx, entity.Checkpoint{RaceID: raceID, Code: "AS2", DisplayName: "AS 2", DisplayOrder: 2})
	require.NoError(t, err)
	_, err = repo.Create(ctx, entity.Checkpoint{RaceID: raceID, Code: "AS1", DisplayName: "AS 1", DisplayOrder: 1})
	require.NoError(t, err)

	cps, err := repo.List(ctx, raceID)
	require.NoError(t, err)
	require.Len(t, cps, 2)
	assert.Equal(t, "AS1", cps[0].Code)
	assert.Equal(t, "AS2", cps[1].Code)
}

func TestCheckpointRepo_Get_NotFound(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewCheckpointRepo(db)

	_, err := repo.Get(context.Background(), 99999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCheckpointRepo_Reorder(t *testing.T) {
	db := requireDB(t)
	raceID := seedRace(t, db, seedEvent(t, db))
	repo := repository.NewCheckpointRepo(db)
	ctx := context.Background()

	cp1, err := repo.Create(ctx, entity.Checkpoint{RaceID: raceID, Code: "A", DisplayName: "A", DisplayOrder: 1})
	require.NoError(t, err)
	cp2, err := repo.Create(ctx, entity.Checkpoint{RaceID: raceID, Code: "B", DisplayName: "B", DisplayOrder: 2})
	require.NoError(t, err)
	cp3, err := repo.Create(ctx, entity.Checkpoint{RaceID: raceID, Code: "C", DisplayName: "C", DisplayOrder: 3})
	require.NoError(t, err)

	// Reverse the order: C, A, B
	err = repo.Reorder(ctx, raceID, []int{cp3.ID, cp1.ID, cp2.ID})
	require.NoError(t, err)

	cps, err := repo.List(ctx, raceID)
	require.NoError(t, err)
	require.Len(t, cps, 3)
	assert.Equal(t, "C", cps[0].Code)
	assert.Equal(t, "A", cps[1].Code)
	assert.Equal(t, "B", cps[2].Code)
}
