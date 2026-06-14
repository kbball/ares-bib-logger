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
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

// logCols matches the SELECT column list used by the log repo.
var logCols = []string{"id", "runner_id", "checkpoint_id", "recorded_at", "source", "raw_message", "created_at"}

func logRow(id, runnerID, checkpointID int, source entity.LogSource, rawMsg interface{}) *sqlmock.Rows {
	return sqlmock.NewRows(logCols).AddRow(id, runnerID, checkpointID, time.Now(), string(source), rawMsg, time.Now())
}

func TestCheckpointLogRepo_Create_WithMessage(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("INSERT INTO checkpoint_logs").
		WithArgs(1, 2, sqlmock.AnyArg(), "MANUAL", "raw").
		WillReturnRows(logRow(10, 1, 2, entity.SourceManual, "raw"))

	log, err := repository.NewCheckpointLogRepo(db).Create(context.Background(), entity.CheckpointLog{
		RunnerID: 1, CheckpointID: 2, RecordedAt: time.Now(),
		Source: entity.SourceManual, RawMessage: "raw",
	})
	require.NoError(t, err)
	assert.Equal(t, 10, log.ID)
	assert.Equal(t, "raw", log.RawMessage)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_Create_NullMessage(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("INSERT INTO checkpoint_logs").
		WithArgs(1, 2, sqlmock.AnyArg(), "MESHTASTIC", nil).
		WillReturnRows(logRow(11, 1, 2, entity.SourceMeshtastic, nil))

	log, err := repository.NewCheckpointLogRepo(db).Create(context.Background(), entity.CheckpointLog{
		RunnerID: 1, CheckpointID: 2, RecordedAt: time.Now(),
		Source: entity.SourceMeshtastic,
	})
	require.NoError(t, err)
	assert.Equal(t, "", log.RawMessage)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_Create_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("INSERT INTO checkpoint_logs").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("fk violation"))

	_, err := repository.NewCheckpointLogRepo(db).Create(context.Background(), entity.CheckpointLog{
		RunnerID: 1, CheckpointID: 2, RecordedAt: time.Now(), Source: entity.SourceManual,
	})
	assert.ErrorContains(t, err, "fk violation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_Upsert_Insert(t *testing.T) {
	db, mock := newMock(t)
	upsertCols := append(logCols, "was_created")

	mock.ExpectQuery("INSERT INTO checkpoint_logs").
		WithArgs(1, 2, sqlmock.AnyArg(), "MANUAL", "raw").
		WillReturnRows(sqlmock.NewRows(upsertCols).
			AddRow(10, 1, 2, time.Now(), "MANUAL", "raw", time.Now(), true))

	log, wasCreated, err := repository.NewCheckpointLogRepo(db).Upsert(context.Background(), entity.CheckpointLog{
		RunnerID: 1, CheckpointID: 2, RecordedAt: time.Now(),
		Source: entity.SourceManual, RawMessage: "raw",
	})
	require.NoError(t, err)
	assert.True(t, wasCreated)
	assert.Equal(t, 10, log.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_Upsert_Update(t *testing.T) {
	db, mock := newMock(t)
	upsertCols := append(logCols, "was_created")

	mock.ExpectQuery("INSERT INTO checkpoint_logs").
		WithArgs(1, 2, sqlmock.AnyArg(), "WINLINK_IMPORT", nil).
		WillReturnRows(sqlmock.NewRows(upsertCols).
			AddRow(10, 1, 2, time.Now(), "WINLINK_IMPORT", nil, time.Now(), false))

	_, wasCreated, err := repository.NewCheckpointLogRepo(db).Upsert(context.Background(), entity.CheckpointLog{
		RunnerID: 1, CheckpointID: 2, RecordedAt: time.Now(),
		Source: entity.SourceWinlinkImport,
	})
	require.NoError(t, err)
	assert.False(t, wasCreated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_Upsert_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("INSERT INTO checkpoint_logs").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("upsert failed"))

	_, _, err := repository.NewCheckpointLogRepo(db).Upsert(context.Background(), entity.CheckpointLog{
		RunnerID: 1, CheckpointID: 2, RecordedAt: time.Now(), Source: entity.SourceManual,
	})
	assert.ErrorContains(t, err, "upsert failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_Exists_True(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT EXISTS(SELECT 1 FROM checkpoint_logs WHERE runner_id = $1 AND checkpoint_id = $2)`)).
		WithArgs(1, 2).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repository.NewCheckpointLogRepo(db).ExistsByRunnerAndCheckpoint(context.Background(), 1, 2)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_Exists_False(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(qe(`SELECT EXISTS(SELECT 1 FROM checkpoint_logs WHERE runner_id = $1 AND checkpoint_id = $2)`)).
		WithArgs(1, 99).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := repository.NewCheckpointLogRepo(db).ExistsByRunnerAndCheckpoint(context.Background(), 1, 99)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_Exists_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("db error"))

	_, err := repository.NewCheckpointLogRepo(db).ExistsByRunnerAndCheckpoint(context.Background(), 1, 2)
	assert.ErrorContains(t, err, "db error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_ListByRaceAndCheckpoint_ReturnsRows(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("SELECT cl.id, cl.runner_id").
		WithArgs(1, 2).
		WillReturnRows(sqlmock.NewRows(logCols).
			AddRow(1, 5, 2, time.Now(), "MANUAL", nil, time.Now()))

	logs, err := repository.NewCheckpointLogRepo(db).ListByRaceAndCheckpoint(context.Background(), 1, 2)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, 5, logs[0].RunnerID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_ListByRaceAndCheckpoint_Error(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("SELECT cl.id, cl.runner_id").
		WithArgs(1, 2).
		WillReturnError(errors.New("query failed"))

	_, err := repository.NewCheckpointLogRepo(db).ListByRaceAndCheckpoint(context.Background(), 1, 2)
	assert.ErrorContains(t, err, "query failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_ListByRace_ReturnsRows(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("SELECT cl.id, cl.runner_id").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(logCols).
			AddRow(1, 5, 2, time.Now(), "MESHTASTIC", "raw payload", time.Now()).
			AddRow(2, 6, 3, time.Now(), "MANUAL", nil, time.Now()))

	logs, err := repository.NewCheckpointLogRepo(db).ListByRace(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, "raw payload", logs[0].RawMessage)
	assert.Equal(t, "", logs[1].RawMessage)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckpointLogRepo_ListByRace_QueryError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery("SELECT cl.id, cl.runner_id").
		WithArgs(1).
		WillReturnError(errors.New("timeout"))

	_, err := repository.NewCheckpointLogRepo(db).ListByRace(context.Background(), 1)
	assert.ErrorContains(t, err, "timeout")
	assert.NoError(t, mock.ExpectationsWereMet())
}
