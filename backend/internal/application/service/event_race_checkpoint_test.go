package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/application/service"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

// --- mockEventRepository ---

type mockEventRepository struct {
	events []entity.Event
}

func (m *mockEventRepository) List(_ context.Context) ([]entity.Event, error) {
	return m.events, nil
}

func (m *mockEventRepository) Get(_ context.Context, id int) (entity.Event, error) {
	for _, e := range m.events {
		if e.ID == id {
			return e, nil
		}
	}
	return entity.Event{}, domain.ErrNotFound
}

func (m *mockEventRepository) Create(_ context.Context, name string) (entity.Event, error) {
	e := entity.Event{ID: len(m.events) + 1, Name: name, CreatedAt: time.Now()}
	m.events = append(m.events, e)
	return e, nil
}

// --- EventService tests ---

func TestEventService_CreateAndList(t *testing.T) {
	repo := &mockEventRepository{}
	svc := service.NewEventService(repo)
	ctx := context.Background()

	e, err := svc.Create(ctx, "GA Death Race")
	require.NoError(t, err)
	assert.Equal(t, "GA Death Race", e.Name)

	events, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, events, 1)

	fetched, err := svc.Get(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, e.ID, fetched.ID)
}

func TestEventService_Get_NotFound(t *testing.T) {
	svc := service.NewEventService(&mockEventRepository{})
	_, err := svc.Get(context.Background(), 99)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- RaceService tests ---

func TestRaceService_Delegates(t *testing.T) {
	races := &mockRaceRepository{
		races: map[int]entity.Race{1: {ID: 1, EventID: 10, Name: "GDR"}},
	}
	svc := service.NewRaceService(races)
	ctx := context.Background()

	list, err := svc.List(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	r, err := svc.Get(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "GDR", r.Name)

	_, err = svc.Create(ctx, 10, "New Race")
	require.NoError(t, err)

	err = svc.Delete(ctx, 1)
	require.NoError(t, err)
}

func TestRaceService_Get_NotFound(t *testing.T) {
	svc := service.NewRaceService(&mockRaceRepository{races: map[int]entity.Race{}})
	_, err := svc.Get(context.Background(), 99)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- CheckpointService tests ---

func TestCheckpointService_CreateAndList(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 5, Code: "AS6"},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{5: {ID: 5}}}
	svc := service.NewCheckpointService(cps, races)
	ctx := context.Background()

	list, err := svc.List(ctx, 5)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	fetched, err := svc.Get(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "AS6", fetched.Code)

	created, err := svc.Create(ctx, entity.Checkpoint{RaceID: 5, Code: "AS7", DisplayName: "AS7", DisplayOrder: 2})
	require.NoError(t, err)
	assert.Equal(t, "AS7", created.Code)
}

func TestCheckpointService_Reorder_Success(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 1}, 2: {ID: 2, RaceID: 1},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, OrderLocked: false}}}
	svc := service.NewCheckpointService(cps, races)

	err := svc.Reorder(context.Background(), 1, []int{2, 1})
	assert.NoError(t, err)
}

func TestCheckpointService_Reorder_Locked(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, OrderLocked: true}}}
	svc := service.NewCheckpointService(cps, races)

	err := svc.Reorder(context.Background(), 1, []int{2, 1})
	assert.ErrorIs(t, err, domain.ErrLocked)
}

func TestCheckpointService_Reorder_RaceNotFound(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}
	races := &mockRaceRepository{races: map[int]entity.Race{}}
	svc := service.NewCheckpointService(cps, races)

	err := svc.Reorder(context.Background(), 999, []int{1, 2})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
