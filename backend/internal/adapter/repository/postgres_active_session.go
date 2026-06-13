package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
)

type ActiveSessionRepo struct {
	db *sql.DB
}

func NewActiveSessionRepo(db *sql.DB) *ActiveSessionRepo { return &ActiveSessionRepo{db: db} }

var _ portrepo.ActiveSessionRepository = (*ActiveSessionRepo)(nil)

func (r *ActiveSessionRepo) Get(ctx context.Context) (entity.ActiveSession, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT s.event_id, asc_.race_id, asc_.checkpoint_id
		 FROM active_session s
		 LEFT JOIN active_session_checkpoints asc_ ON asc_.session_id = s.id
		 WHERE s.id = 1`)
	if err != nil {
		return entity.ActiveSession{}, fmt.Errorf("querying active session: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sess entity.ActiveSession
	first := true
	for rows.Next() {
		var eventID sql.NullInt64
		var raceID sql.NullInt64
		var checkpointID sql.NullInt64
		if err := rows.Scan(&eventID, &raceID, &checkpointID); err != nil {
			return entity.ActiveSession{}, fmt.Errorf("scanning active session: %w", err)
		}
		if first {
			if eventID.Valid {
				id := int(eventID.Int64)
				sess.EventID = &id
			}
			first = false
		}
		if raceID.Valid {
			sess.Checkpoints = append(sess.Checkpoints, entity.ActiveSessionCheckpoint{
				RaceID:       int(raceID.Int64),
				CheckpointID: int(checkpointID.Int64),
			})
		}
	}
	return sess, rows.Err()
}

func (r *ActiveSessionRepo) SetEvent(ctx context.Context, eventID int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE active_session SET event_id = $1, updated_at = NOW() WHERE id = 1`, eventID)
	return err
}

func (r *ActiveSessionRepo) SetCheckpoint(ctx context.Context, raceID, checkpointID int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO active_session_checkpoints (session_id, race_id, checkpoint_id)
		 VALUES (1, $1, $2)
		 ON CONFLICT (session_id, race_id) DO UPDATE SET checkpoint_id = EXCLUDED.checkpoint_id`,
		raceID, checkpointID)
	return err
}

func (r *ActiveSessionRepo) ClearCheckpoint(ctx context.Context, raceID int) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM active_session_checkpoints WHERE session_id = 1 AND race_id = $1`, raceID)
	return err
}
