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

func (m *mockEventRepository) Archive(_ context.Context, id int) error {
	return nil
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

// --- EventService.Archive ---

func TestEventService_Archive(t *testing.T) {
	svc := service.NewEventService(&mockEventRepository{})
	err := svc.Archive(context.Background(), 1)
	assert.NoError(t, err)
}

// --- RaceService.LockOrder ---

func TestRaceService_LockOrder(t *testing.T) {
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1}}}
	svc := service.NewRaceService(races)
	err := svc.LockOrder(context.Background(), 1)
	assert.NoError(t, err)
}

// --- CheckpointService.Create auto-order ---

func TestCheckpointService_Create_AutoOrder(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 1, DisplayOrder: 1},
		2: {ID: 2, RaceID: 1, DisplayOrder: 2},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1}}}
	svc := service.NewCheckpointService(cps, races)

	created, err := svc.Create(context.Background(), entity.Checkpoint{
		RaceID: 1, Code: "AS3", DisplayName: "Aid 3", DisplayOrder: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, created.DisplayOrder)
}

func TestCheckpointService_Create_AutoOrder_ListError(t *testing.T) {
	cps := &mockCheckpointRepository{
		checkpoints: map[int]entity.Checkpoint{},
		listErr:     errors.New("db failure"),
	}
	races := &mockRaceRepository{races: map[int]entity.Race{}}
	svc := service.NewCheckpointService(cps, races)

	_, err := svc.Create(context.Background(), entity.Checkpoint{
		RaceID: 1, Code: "AS3", DisplayName: "Aid 3", DisplayOrder: 0,
	})
	assert.ErrorContains(t, err, "db failure")
}

// --- CheckpointService.Update ---

func TestCheckpointService_Update_Success(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 5, Code: "AS1", DisplayName: "Aid 1"},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{5: {ID: 5, OrderLocked: false}}}
	svc := service.NewCheckpointService(cps, races)

	dist := 5.0
	updated, err := svc.Update(context.Background(), 1, "AS1-NEW", "Aid 1 Updated", &dist)
	require.NoError(t, err)
	assert.Equal(t, "AS1-NEW", updated.Code)
}

func TestCheckpointService_Update_CPNotFound(t *testing.T) {
	cps := &mockCheckpointRepository{getErr: domain.ErrNotFound}
	svc := service.NewCheckpointService(cps, &mockRaceRepository{races: map[int]entity.Race{}})

	_, err := svc.Update(context.Background(), 99, "X", "Y", nil)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCheckpointService_Update_RaceNotFound(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 999},
	}}
	svc := service.NewCheckpointService(cps, &mockRaceRepository{races: map[int]entity.Race{}})

	_, err := svc.Update(context.Background(), 1, "X", "Y", nil)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCheckpointService_Update_Locked(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 5},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{5: {ID: 5, OrderLocked: true}}}
	svc := service.NewCheckpointService(cps, races)

	_, err := svc.Update(context.Background(), 1, "X", "Y", nil)
	assert.ErrorIs(t, err, domain.ErrLocked)
}

// --- CheckpointService.Delete ---

func TestCheckpointService_Delete_Success(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 5},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{5: {ID: 5, OrderLocked: false}}}
	svc := service.NewCheckpointService(cps, races)

	err := svc.Delete(context.Background(), 1)
	assert.NoError(t, err)
}

func TestCheckpointService_Delete_CPNotFound(t *testing.T) {
	cps := &mockCheckpointRepository{getErr: domain.ErrNotFound}
	svc := service.NewCheckpointService(cps, &mockRaceRepository{races: map[int]entity.Race{}})

	err := svc.Delete(context.Background(), 99)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCheckpointService_Delete_RaceNotFound(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 999},
	}}
	svc := service.NewCheckpointService(cps, &mockRaceRepository{races: map[int]entity.Race{}})

	err := svc.Delete(context.Background(), 1)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCheckpointService_Delete_Locked(t *testing.T) {
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{
		1: {ID: 1, RaceID: 5},
	}}
	races := &mockRaceRepository{races: map[int]entity.Race{5: {ID: 5, OrderLocked: true}}}
	svc := service.NewCheckpointService(cps, races)

	err := svc.Delete(context.Background(), 1)
	assert.ErrorIs(t, err, domain.ErrLocked)
}
