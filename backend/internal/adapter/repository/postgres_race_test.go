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

var raceCols = []string{"id", "event_id", "name", "roster_locked", "order_locked", "created_at"}

func raceRow(id, eventID int, name string, rosterLocked, orderLocked bool) *sqlmock.Rows {
	return sqlmock.NewRows(raceCols).AddRow(id, eventID, name, rosterLocked, orderLocked, time.Now())
}

func TestRaceRepo_List_ReturnsRows(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, event_id, name, roster_locked, order_locked, created_at FROM races WHERE event_id = $1 ORDER BY created_at`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(raceCols).
			AddRow(1, 1, "GDR", false, false, time.Now()).
			AddRow(2, 1, "50M", false, false, time.Now()))

	races, err := repository.NewRaceRepo(db).List(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, races, 2)
	assert.Equal(t, "GDR", races[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRaceRepo_List_Empty(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, event_id, name, roster_locked, order_locked, created_at FROM races WHERE event_id = $1 ORDER BY created_at`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(raceCols))

	races, err := repository.NewRaceRepo(db).List(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, races)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRaceRepo_List_QueryError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, event_id, name, roster_locked, order_locked, created_at FROM races WHERE event_id = $1 ORDER BY created_at`)).
		WithArgs(1).
		WillReturnError(errors.New("db error"))

	_, err := repository.NewRaceRepo(db).List(context.Background(), 1)
	assert.ErrorContains(t, err, "db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRaceRepo_Get_Found(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, event_id, name, roster_locked, order_locked, created_at FROM races WHERE id = $1`)).
		WithArgs(7).
		WillReturnRows(raceRow(7, 1, "GDR", false, true))

	race, err := repository.NewRaceRepo(db).Get(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, 7, race.ID)
	assert.True(t, race.OrderLocked)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRaceRepo_Get_NotFound(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, event_id, name, roster_locked, order_locked, created_at FROM races WHERE id = $1`)).
		WithArgs(999).
		WillReturnRows(sqlmock.NewRows(raceCols))

	_, err := repository.NewRaceRepo(db).Get(context.Background(), 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRaceRepo_Create_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe("INSERT INTO races (event_id, name) VALUES ($1, $2)\n\t\t\t RETURNING id, event_id, name, roster_locked, order_locked, created_at")).
		WithArgs(1, "50M").
		WillReturnRows(raceRow(3, 1, "50M", false, false))

	race, err := repository.NewRaceRepo(db).Create(context.Background(), 1, "50M")
	require.NoError(t, err)
	assert.Equal(t, "50M", race.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRaceRepo_Create_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("INSERT INTO races").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("fk violation"))

	_, err := repository.NewRaceRepo(db).Create(context.Background(), 99, "Bad Race")
	assert.ErrorContains(t, err, "fk violation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRaceRepo_LockRoster_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`UPDATE races SET roster_locked = true WHERE id = $1`)).
		WithArgs(4).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repository.NewRaceRepo(db).LockRoster(context.Background(), 4)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRaceRepo_LockOrder_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`UPDATE races SET order_locked = true WHERE id = $1`)).
		WithArgs(4).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repository.NewRaceRepo(db).LockOrder(context.Background(), 4)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRaceRepo_Delete_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`DELETE FROM races WHERE id = $1`)).
		WithArgs(4).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repository.NewRaceRepo(db).Delete(context.Background(), 4)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
