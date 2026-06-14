package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/repository"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
)

var eventCols = []string{"id", "name", "archived", "created_at"}

func TestEventRepo_List_ReturnsRows(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()

	mock.ExpectQuery(qe(`SELECT id, name, archived, created_at FROM events WHERE NOT archived ORDER BY created_at DESC`)).
		WillReturnRows(sqlmock.NewRows(eventCols).
			AddRow(1, "Event A", false, now).
			AddRow(2, "Event B", false, now))

	events, err := repository.NewEventRepo(db).List(context.Background())
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "Event A", events[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepo_List_Empty(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, name, archived, created_at FROM events WHERE NOT archived ORDER BY created_at DESC`)).
		WillReturnRows(sqlmock.NewRows(eventCols))

	events, err := repository.NewEventRepo(db).List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, events)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepo_List_QueryError(t *testing.T) {
	db, mock := newMock(t)
	dbErr := errors.New("connection reset")

	mock.ExpectQuery(qe(`SELECT id, name, archived, created_at FROM events WHERE NOT archived ORDER BY created_at DESC`)).
		WillReturnError(dbErr)

	_, err := repository.NewEventRepo(db).List(context.Background())
	assert.ErrorContains(t, err, "connection reset")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepo_Get_Found(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()

	mock.ExpectQuery(qe(`SELECT id, name, archived, created_at FROM events WHERE id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(eventCols).AddRow(1, "GA Death Race", false, now))

	event, err := repository.NewEventRepo(db).Get(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, event.ID)
	assert.Equal(t, "GA Death Race", event.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepo_Get_NotFound(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, name, archived, created_at FROM events WHERE id = $1`)).
		WithArgs(999).
		WillReturnRows(sqlmock.NewRows(eventCols)) // no rows

	_, err := repository.NewEventRepo(db).Get(context.Background(), 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepo_Get_DBError(t *testing.T) {
	db, mock := newMock(t)
	dbErr := errors.New("timeout")

	mock.ExpectQuery(qe(`SELECT id, name, archived, created_at FROM events WHERE id = $1`)).
		WithArgs(1).
		WillReturnError(dbErr)

	_, err := repository.NewEventRepo(db).Get(context.Background(), 1)
	assert.ErrorIs(t, err, dbErr)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepo_Create_Success(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()

	mock.ExpectQuery(qe(`INSERT INTO events (name) VALUES ($1) RETURNING id, name, archived, created_at`)).
		WithArgs("New Event").
		WillReturnRows(sqlmock.NewRows(eventCols).AddRow(5, "New Event", false, now))

	event, err := repository.NewEventRepo(db).Create(context.Background(), "New Event")
	require.NoError(t, err)
	assert.Equal(t, 5, event.ID)
	assert.Equal(t, "New Event", event.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepo_Create_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`INSERT INTO events (name) VALUES ($1) RETURNING id, name, archived, created_at`)).
		WithArgs("Duplicate").
		WillReturnError(errors.New("unique constraint violation"))

	_, err := repository.NewEventRepo(db).Create(context.Background(), "Duplicate")
	assert.ErrorContains(t, err, "unique constraint violation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepo_Archive_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`UPDATE events SET archived = true WHERE id = $1`)).
		WithArgs(3).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repository.NewEventRepo(db).Archive(context.Background(), 3)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepo_Archive_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`UPDATE events SET archived = true WHERE id = $1`)).
		WithArgs(3).
		WillReturnError(errors.New("db down"))

	err := repository.NewEventRepo(db).Archive(context.Background(), 3)
	assert.ErrorContains(t, err, "db down")
	assert.NoError(t, mock.ExpectationsWereMet())
}
