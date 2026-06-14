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

var cpCols = []string{"id", "race_id", "code", "display_name", "display_order", "distance_from_start", "created_at"}

func cpRow(id, raceID int, code, name string, order int, dist interface{}) *sqlmock.Rows {
	return sqlmock.NewRows(cpCols).AddRow(id, raceID, code, name, order, dist, time.Now())
}

func TestCheckpointRepo_List_ReturnsRows(t *testing.T) {
	db, mock := newMock(t)
	dist := 5.5

	mock.ExpectQuery(qe(`SELECT id, race_id, code, display_name, display_order, distance_from_start, created_at FROM checkpoints WHERE race_id = $1 ORDER BY display_order`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(cpCols).
			AddRow(1, 1, "AS1", "Aid 1", 1, nil, time.Now()).
			AddRow(2, 1, "AS2", "Aid 2", 2, &dist, time.Now()))

	cps, err := repository.NewCheckpointRepo(db).List(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, cps, 2)
	assert.Nil(t, cps[0].DistanceFromStart)
	assert.InDelta(t, 5.5, *cps[1].DistanceFromStart, 0.001)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_List_QueryError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, race_id, code, display_name, display_order, distance_from_start, created_at FROM checkpoints WHERE race_id = $1 ORDER BY display_order`)).
		WithArgs(1).
		WillReturnError(errors.New("db error"))

	_, err := repository.NewCheckpointRepo(db).List(context.Background(), 1)
	assert.ErrorContains(t, err, "db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Get_Found(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, race_id, code, display_name, display_order, distance_from_start, created_at FROM checkpoints WHERE id = $1`)).
		WithArgs(10).
		WillReturnRows(cpRow(10, 1, "AS1", "Aid 1", 1, nil))

	cp, err := repository.NewCheckpointRepo(db).Get(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 10, cp.ID)
	assert.Equal(t, "AS1", cp.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Get_NotFound(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT id, race_id, code, display_name, display_order, distance_from_start, created_at FROM checkpoints WHERE id = $1`)).
		WithArgs(999).
		WillReturnRows(sqlmock.NewRows(cpCols))

	_, err := repository.NewCheckpointRepo(db).Get(context.Background(), 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Create_Success(t *testing.T) {
	db, mock := newMock(t)
	dist := 10.0

	mock.ExpectQuery("INSERT INTO checkpoints").
		WithArgs(1, "AS3", "Aid 3", 3, &dist).
		WillReturnRows(cpRow(3, 1, "AS3", "Aid 3", 3, &dist))

	cp, err := repository.NewCheckpointRepo(db).Create(context.Background(), entity.Checkpoint{
		RaceID: 1, Code: "AS3", DisplayName: "Aid 3", DisplayOrder: 3,
		DistanceFromStart: &dist,
	})
	require.NoError(t, err)
	assert.Equal(t, "AS3", cp.Code)
	assert.InDelta(t, 10.0, *cp.DistanceFromStart, 0.001)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Create_NoDistance(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("INSERT INTO checkpoints").
		WithArgs(1, "AS3", "Aid 3", 3, nil).
		WillReturnRows(cpRow(3, 1, "AS3", "Aid 3", 3, nil))

	cp, err := repository.NewCheckpointRepo(db).Create(context.Background(), entity.Checkpoint{
		RaceID: 1, Code: "AS3", DisplayName: "Aid 3", DisplayOrder: 3,
	})
	require.NoError(t, err)
	assert.Nil(t, cp.DistanceFromStart)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Create_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("INSERT INTO checkpoints").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("unique violation"))

	_, err := repository.NewCheckpointRepo(db).Create(context.Background(), entity.Checkpoint{
		RaceID: 1, Code: "AS3", DisplayName: "Aid 3", DisplayOrder: 3,
	})
	assert.ErrorContains(t, err, "unique violation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Update_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("UPDATE checkpoints SET code").
		WithArgs("AS1-NEW", "Aid 1 Updated", nil, 10).
		WillReturnRows(cpRow(10, 1, "AS1-NEW", "Aid 1 Updated", 1, nil))

	updated, err := repository.NewCheckpointRepo(db).Update(context.Background(), entity.Checkpoint{
		ID: 10, Code: "AS1-NEW", DisplayName: "Aid 1 Updated",
	})
	require.NoError(t, err)
	assert.Equal(t, "AS1-NEW", updated.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Update_NotFound(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("UPDATE checkpoints SET code").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 999).
		WillReturnRows(sqlmock.NewRows(cpCols))

	_, err := repository.NewCheckpointRepo(db).Update(context.Background(), entity.Checkpoint{ID: 999})
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Delete_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectExec(qe(`DELETE FROM checkpoints WHERE id = $1`)).
		WithArgs(10).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repository.NewCheckpointRepo(db).Delete(context.Background(), 10)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Reorder_Success(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(qe(`UPDATE checkpoints SET display_order = display_order + 10000 WHERE race_id = $1`)).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec("UPDATE checkpoints AS c SET display_order").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	err := repository.NewCheckpointRepo(db).Reorder(context.Background(), 1, []int{3, 1, 2})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Reorder_BeginError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err := repository.NewCheckpointRepo(db).Reorder(context.Background(), 1, []int{1, 2})
	assert.ErrorContains(t, err, "begin failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Reorder_ShiftError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(qe(`UPDATE checkpoints SET display_order = display_order + 10000 WHERE race_id = $1`)).
		WithArgs(1).
		WillReturnError(errors.New("shift failed"))
	mock.ExpectRollback()

	err := repository.NewCheckpointRepo(db).Reorder(context.Background(), 1, []int{1, 2})
	assert.ErrorContains(t, err, "shift failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointRepo_Reorder_ApplyError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(qe(`UPDATE checkpoints SET display_order = display_order + 10000 WHERE race_id = $1`)).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE checkpoints AS c SET display_order").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnError(errors.New("apply failed"))
	mock.ExpectRollback()

	err := repository.NewCheckpointRepo(db).Reorder(context.Background(), 1, []int{1, 2})
	assert.ErrorContains(t, err, "apply failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}
