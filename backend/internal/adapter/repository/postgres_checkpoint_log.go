package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
)

type CheckpointLogRepo struct {
	db *sql.DB
}

func NewCheckpointLogRepo(db *sql.DB) *CheckpointLogRepo { return &CheckpointLogRepo{db: db} }

var _ portrepo.CheckpointLogRepository = (*CheckpointLogRepo)(nil)

const logCols = `id, runner_id, checkpoint_id, recorded_at, source, raw_message, created_at`

func scanLog(s interface{ Scan(...any) error }) (entity.CheckpointLog, error) {
	var l entity.CheckpointLog
	var rawMsg sql.NullString
	err := s.Scan(&l.ID, &l.RunnerID, &l.CheckpointID, &l.RecordedAt, &l.Source, &rawMsg, &l.CreatedAt)
	if err == nil {
		l.RawMessage = rawMsg.String
	}
	return l, err
}

func (r *CheckpointLogRepo) Create(ctx context.Context, log entity.CheckpointLog) (entity.CheckpointLog, error) {
	created, err := scanLog(r.db.QueryRowContext(ctx,
		`INSERT INTO checkpoint_logs (runner_id, checkpoint_id, recorded_at, source, raw_message)
		 VALUES ($1, $2, $3, $4, $5) RETURNING `+logCols,
		log.RunnerID, log.CheckpointID, log.RecordedAt, string(log.Source), nullableStr(log.RawMessage)))
	if err != nil {
		return entity.CheckpointLog{}, fmt.Errorf("creating checkpoint log: %w", err)
	}
	return created, nil
}

func (r *CheckpointLogRepo) Upsert(ctx context.Context, log entity.CheckpointLog) (entity.CheckpointLog, bool, error) {
	var wasCreated bool
	var l entity.CheckpointLog
	var rawMsg sql.NullString
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO checkpoint_logs (runner_id, checkpoint_id, recorded_at, source, raw_message)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (runner_id, checkpoint_id) DO UPDATE
		   SET recorded_at = EXCLUDED.recorded_at,
		       source      = EXCLUDED.source,
		       raw_message = EXCLUDED.raw_message
		 RETURNING `+logCols+`, (xmax = 0)`,
		log.RunnerID, log.CheckpointID, log.RecordedAt, string(log.Source), nullableStr(log.RawMessage),
	).Scan(&l.ID, &l.RunnerID, &l.CheckpointID, &l.RecordedAt, &l.Source, &rawMsg, &l.CreatedAt, &wasCreated)
	if err != nil {
		return entity.CheckpointLog{}, false, fmt.Errorf("upserting checkpoint log: %w", err)
	}
	l.RawMessage = rawMsg.String
	return l, wasCreated, nil
}

func (r *CheckpointLogRepo) ExistsByRunnerAndCheckpoint(ctx context.Context, runnerID, checkpointID int) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM checkpoint_logs WHERE runner_id = $1 AND checkpoint_id = $2)`,
		runnerID, checkpointID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking log existence: %w", err)
	}
	return exists, nil
}

func (r *CheckpointLogRepo) ListByRaceAndCheckpoint(ctx context.Context, raceID, checkpointID int) ([]entity.CheckpointLog, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+logColsAliased("cl")+`
		 FROM checkpoint_logs cl
		 JOIN runners run ON run.id = cl.runner_id
		 WHERE run.race_id = $1 AND cl.checkpoint_id = $2
		 ORDER BY cl.recorded_at`,
		raceID, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("listing checkpoint logs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanLogs(rows)
}

func (r *CheckpointLogRepo) ListByRace(ctx context.Context, raceID int) ([]entity.CheckpointLog, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+logColsAliased("cl")+`
		 FROM checkpoint_logs cl
		 JOIN runners run ON run.id = cl.runner_id
		 WHERE run.race_id = $1
		 ORDER BY cl.recorded_at`,
		raceID)
	if err != nil {
		return nil, fmt.Errorf("listing checkpoint logs by race: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanLogs(rows)
}

func scanLogs(rows *sql.Rows) ([]entity.CheckpointLog, error) {
	var logs []entity.CheckpointLog
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning checkpoint log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// logColsAliased returns the log column list with a table alias prefix (e.g. "cl.id, cl.runner_id, ...").
func logColsAliased(alias string) string {
	return alias + ".id, " + alias + ".runner_id, " + alias + ".checkpoint_id, " +
		alias + ".recorded_at, " + alias + ".source, " + alias + ".raw_message, " + alias + ".created_at"
}

func nullableStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
