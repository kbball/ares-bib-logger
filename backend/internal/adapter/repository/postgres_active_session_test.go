package repository_test

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/repository"
)

var sessionCols = []string{"event_id", "race_id", "checkpoint_id"}

func TestActiveSessionRepo_Get_Empty(t *testing.T) {
	db, mock := newMock(t)

	// No rows returned: session exists but has no event and no checkpoints.
	mock.ExpectQuery("SELECT s.event_id, asc_.race_id, asc_.checkpoint_id").
		WillReturnRows(sqlmock.NewRows(sessionCols))

	sess, err := repository.NewActiveSessionRepo(db).Get(context.Background())
	require.NoError(t, err)
	assert.Nil(t, sess.EventID)
	assert.Empty(t, sess.Checkpoints)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveSessionRepo_Get_WithEventNoCheckpoints(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("SELECT s.event_id, asc_.race_id, asc_.checkpoint_id").
		WillReturnRows(sqlmock.NewRows(sessionCols).
			AddRow(3, nil, nil)) // event set, no checkpoint row

	sess, err := repository.NewActiveSessionRepo(db).Get(context.Background())
	require.NoError(t, err)
	require.NotNil(t, sess.EventID)
	assert.Equal(t, 3, *sess.EventID)
	assert.Empty(t, sess.Checkpoints)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveSessionRepo_Get_WithCheckpoints(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("SELECT s.event_id, asc_.race_id, asc_.checkpoint_id").
		WillReturnRows(sqlmock.NewRows(sessionCols).
			AddRow(1, 10, 100).
			AddRow(1, 20, 200))

	sess, err := repository.NewActiveSessionRepo(db).Get(context.Background())
	require.NoError(t, err)
	require.NotNil(t, sess.EventID)
	assert.Equal(t, 1, *sess.EventID)
	assert.Len(t, sess.Checkpoints, 2)
	assert.Equal(t, 10, sess.Checkpoints[0].RaceID)
	assert.Equal(t, 100, sess.Checkpoints[0].CheckpointID)
	assert.Equal(t, 20, sess.Checkpoints[1].RaceID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveSessionRepo_Get_QueryError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("SELECT s.event_id, asc_.race_id, asc_.checkpoint_id").
		WillReturnError(errors.New("db down"))

	_, err := repository.NewActiveSessionRepo(db).Get(context.Background())
	assert.ErrorContains(t, err, "db down")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveSessionRepo_SetEvent_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`UPDATE active_session SET event_id = $1, updated_at = NOW() WHERE id = 1`)).
		WithArgs(5).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repository.NewActiveSessionRepo(db).SetEvent(context.Background(), 5)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveSessionRepo_SetEvent_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`UPDATE active_session SET event_id = $1, updated_at = NOW() WHERE id = 1`)).
		WithArgs(99).
		WillReturnError(errors.New("fk violation"))

	err := repository.NewActiveSessionRepo(db).SetEvent(context.Background(), 99)
	assert.ErrorContains(t, err, "fk violation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveSessionRepo_SetCheckpoint_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec("INSERT INTO active_session_checkpoints").
		WithArgs(2, 10).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repository.NewActiveSessionRepo(db).SetCheckpoint(context.Background(), 2, 10)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveSessionRepo_SetCheckpoint_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec("INSERT INTO active_session_checkpoints").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("conflict"))

	err := repository.NewActiveSessionRepo(db).SetCheckpoint(context.Background(), 2, 10)
	assert.ErrorContains(t, err, "conflict")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveSessionRepo_ClearCheckpoint_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`DELETE FROM active_session_checkpoints WHERE session_id = 1 AND race_id = $1`)).
		WithArgs(2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repository.NewActiveSessionRepo(db).ClearCheckpoint(context.Background(), 2)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActiveSessionRepo_ClearCheckpoint_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`DELETE FROM active_session_checkpoints WHERE session_id = 1 AND race_id = $1`)).
		WithArgs(2).
		WillReturnError(errors.New("db error"))

	err := repository.NewActiveSessionRepo(db).ClearCheckpoint(context.Background(), 2)
	assert.ErrorContains(t, err, "db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}
