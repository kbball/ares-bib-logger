package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/application/service"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

func TestSessionService_Get(t *testing.T) {
	eventIDVal := 5
	repo := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID: &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{
			{RaceID: 1, CheckpointID: 9},
		},
	}}

	svc := service.NewSessionService(repo)
	sess, err := svc.Get(context.Background())

	require.NoError(t, err)
	require.NotNil(t, sess.EventID)
	assert.Equal(t, 5, *sess.EventID)
	assert.Len(t, sess.Checkpoints, 1)
}

func TestSessionService_SetEvent(t *testing.T) {
	repo := &mockActiveSessionRepository{}
	svc := service.NewSessionService(repo)

	err := svc.SetEvent(context.Background(), 42)

	require.NoError(t, err)
	require.NotNil(t, repo.session.EventID)
	assert.Equal(t, 42, *repo.session.EventID)
}

func TestSessionService_SetCheckpoint_AddsNew(t *testing.T) {
	repo := &mockActiveSessionRepository{}
	svc := service.NewSessionService(repo)

	err := svc.SetCheckpoint(context.Background(), 1, 7)

	require.NoError(t, err)
	require.Len(t, repo.session.Checkpoints, 1)
	assert.Equal(t, 1, repo.session.Checkpoints[0].RaceID)
	assert.Equal(t, 7, repo.session.Checkpoints[0].CheckpointID)
}

func TestSessionService_SetCheckpoint_UpdatesExisting(t *testing.T) {
	repo := &mockActiveSessionRepository{session: entity.ActiveSession{
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 7}},
	}}
	svc := service.NewSessionService(repo)

	err := svc.SetCheckpoint(context.Background(), 1, 12)

	require.NoError(t, err)
	require.Len(t, repo.session.Checkpoints, 1)
	assert.Equal(t, 12, repo.session.Checkpoints[0].CheckpointID)
}

func TestSessionService_ClearCheckpoint(t *testing.T) {
	repo := &mockActiveSessionRepository{session: entity.ActiveSession{
		Checkpoints: []entity.ActiveSessionCheckpoint{
			{RaceID: 1, CheckpointID: 7},
			{RaceID: 2, CheckpointID: 8},
		},
	}}
	svc := service.NewSessionService(repo)

	err := svc.ClearCheckpoint(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, repo.session.Checkpoints, 1)
	assert.Equal(t, 2, repo.session.Checkpoints[0].RaceID)
}
