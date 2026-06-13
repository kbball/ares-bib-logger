package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/repository"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
)

func TestEventRepo_CreateAndGet(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewEventRepo(db)
	ctx := context.Background()

	event, err := repo.Create(ctx, "GA Death Race 2026")
	require.NoError(t, err)
	assert.Greater(t, event.ID, 0)
	assert.Equal(t, "GA Death Race 2026", event.Name)

	fetched, err := repo.Get(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, event.ID, fetched.ID)
	assert.Equal(t, event.Name, fetched.Name)
}

func TestEventRepo_List(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewEventRepo(db)
	ctx := context.Background()

	_, err := repo.Create(ctx, "Event A")
	require.NoError(t, err)
	_, err = repo.Create(ctx, "Event B")
	require.NoError(t, err)

	events, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestEventRepo_Get_NotFound(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewEventRepo(db)

	_, err := repo.Get(context.Background(), 99999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestEventRepo_Create_DuplicateNameErrors(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewEventRepo(db)
	ctx := context.Background()

	_, err := repo.Create(ctx, "Unique Event")
	require.NoError(t, err)

	_, err = repo.Create(ctx, "Unique Event")
	assert.Error(t, err)
}
