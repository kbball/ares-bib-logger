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
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

var runnerCols = []string{"id", "race_id", "bib_number", "first_name", "last_name", "sort_order", "status", "created_at", "updated_at"}

func runnerRow(id, raceID, bib int, first, last string, order int, status entity.RunnerStatus) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows(runnerCols).AddRow(id, raceID, bib, first, last, order, string(status), now, now)
}

func TestRunnerRepo_List_ReturnsRows(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()

	mock.ExpectQuery(qe(`SELECT id, race_id, bib_number, first_name, last_name, sort_order, status, created_at, updated_at FROM runners WHERE race_id = $1 ORDER BY sort_order`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(runnerCols).
			AddRow(1, 1, 100, "Alice", "Smith", 1, "ACTIVE", now, now).
			AddRow(2, 1, 101, "Bob", "Jones", 2, "UNKNOWN", now, now))

	runners, err := repository.NewRunnerRepo(db).List(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, runners, 2)
	assert.Equal(t, 100, runners[0].BibNumber)
	assert.Equal(t, entity.StatusActive, runners[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_List_QueryError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, race_id, bib_number, first_name, last_name, sort_order, status, created_at, updated_at FROM runners WHERE race_id = $1 ORDER BY sort_order`)).
		WithArgs(1).
		WillReturnError(errors.New("db error"))

	_, err := repository.NewRunnerRepo(db).List(context.Background(), 1)
	assert.ErrorContains(t, err, "db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_Get_Found(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, race_id, bib_number, first_name, last_name, sort_order, status, created_at, updated_at FROM runners WHERE id = $1`)).
		WithArgs(5).
		WillReturnRows(runnerRow(5, 1, 100, "Alice", "Smith", 1, entity.StatusActive))

	runner, err := repository.NewRunnerRepo(db).Get(context.Background(), 5)
	require.NoError(t, err)
	assert.Equal(t, 5, runner.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_Get_NotFound(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, race_id, bib_number, first_name, last_name, sort_order, status, created_at, updated_at FROM runners WHERE id = $1`)).
		WithArgs(999).
		WillReturnRows(sqlmock.NewRows(runnerCols))

	_, err := repository.NewRunnerRepo(db).Get(context.Background(), 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_GetByBibInEvent_Found(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()

	mock.ExpectQuery("SELECT r.id, r.race_id").
		WithArgs(1, 100).
		WillReturnRows(sqlmock.NewRows(runnerCols).
			AddRow(5, 1, 100, "Alice", "Smith", 1, "ACTIVE", now, now))

	runner, err := repository.NewRunnerRepo(db).GetByBibInEvent(context.Background(), 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 100, runner.BibNumber)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_GetByBibInEvent_NotFound(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("SELECT r.id, r.race_id").
		WithArgs(1, 999).
		WillReturnRows(sqlmock.NewRows(runnerCols))

	_, err := repository.NewRunnerRepo(db).GetByBibInEvent(context.Background(), 1, 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_BulkCreate_Empty(t *testing.T) {
	db, _ := newMock(t)

	// No DB call expected for empty slice.
	err := repository.NewRunnerRepo(db).BulkCreate(context.Background(), nil)
	assert.NoError(t, err)
}

func TestRunnerRepo_BulkCreate_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec("INSERT INTO runners").
		WithArgs(1, 100, "Alice", "Smith", 1, "UNKNOWN",
			1, 101, "Bob", "Jones", 2, "UNKNOWN").
		WillReturnResult(sqlmock.NewResult(0, 2))

	runners := []entity.Runner{
		{RaceID: 1, BibNumber: 100, FirstName: "Alice", LastName: "Smith", SortOrder: 1, Status: entity.StatusUnknown},
		{RaceID: 1, BibNumber: 101, FirstName: "Bob", LastName: "Jones", SortOrder: 2, Status: entity.StatusUnknown},
	}
	err := repository.NewRunnerRepo(db).BulkCreate(context.Background(), runners)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_BulkCreate_ExecError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec("INSERT INTO runners").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("constraint violation"))

	runners := []entity.Runner{
		{RaceID: 1, BibNumber: 100, FirstName: "Alice", LastName: "Smith", SortOrder: 1, Status: entity.StatusUnknown},
	}
	err := repository.NewRunnerRepo(db).BulkCreate(context.Background(), runners)
	assert.ErrorContains(t, err, "constraint violation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_UpdateStatus_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`UPDATE runners SET status = $1, updated_at = NOW() WHERE id = $2`)).
		WithArgs("DNS", 5).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repository.NewRunnerRepo(db).UpdateStatus(context.Background(), 5, entity.StatusDNS)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_MaxSortOrder_ReturnsValue(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT COALESCE(MAX(sort_order), 0) FROM runners WHERE race_id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(7))

	max, err := repository.NewRunnerRepo(db).MaxSortOrder(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 7, max)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_MaxSortOrder_Zero(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT COALESCE(MAX(sort_order), 0) FROM runners WHERE race_id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))

	max, err := repository.NewRunnerRepo(db).MaxSortOrder(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 0, max)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunnerRepo_MaxSortOrder_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT COALESCE(MAX(sort_order), 0) FROM runners WHERE race_id = $1`)).
		WithArgs(1).
		WillReturnError(errors.New("db error"))

	_, err := repository.NewRunnerRepo(db).MaxSortOrder(context.Background(), 1)
	assert.ErrorContains(t, err, "db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}
