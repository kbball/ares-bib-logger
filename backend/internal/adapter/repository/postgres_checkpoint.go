package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
)

type CheckpointRepo struct {
	db *sql.DB
}

func NewCheckpointRepo(db *sql.DB) *CheckpointRepo { return &CheckpointRepo{db: db} }

var _ portrepo.CheckpointRepository = (*CheckpointRepo)(nil)

const cpCols = `id, race_id, code, display_name, display_order, created_at`

func scanCheckpoint(s interface{ Scan(...any) error }) (entity.Checkpoint, error) {
	var cp entity.Checkpoint
	err := s.Scan(&cp.ID, &cp.RaceID, &cp.Code, &cp.DisplayName, &cp.DisplayOrder, &cp.CreatedAt)
	return cp, err
}

func (r *CheckpointRepo) List(ctx context.Context, raceID int) ([]entity.Checkpoint, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+cpCols+` FROM checkpoints WHERE race_id = $1 ORDER BY display_order`, raceID)
	if err != nil {
		return nil, fmt.Errorf("listing checkpoints: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var cps []entity.Checkpoint
	for rows.Next() {
		cp, err := scanCheckpoint(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning checkpoint: %w", err)
		}
		cps = append(cps, cp)
	}
	return cps, rows.Err()
}

func (r *CheckpointRepo) Get(ctx context.Context, id int) (entity.Checkpoint, error) {
	cp, err := scanCheckpoint(r.db.QueryRowContext(ctx,
		`SELECT `+cpCols+` FROM checkpoints WHERE id = $1`, id))
	if err != nil {
		return entity.Checkpoint{}, mapNotFound(err)
	}
	return cp, nil
}

func (r *CheckpointRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM checkpoints WHERE id = $1`, id)
	return err
}

func (r *CheckpointRepo) Create(ctx context.Context, cp entity.Checkpoint) (entity.Checkpoint, error) {
	created, err := scanCheckpoint(r.db.QueryRowContext(ctx,
		`INSERT INTO checkpoints (race_id, code, display_name, display_order)
		 VALUES ($1, $2, $3, $4) RETURNING `+cpCols,
		cp.RaceID, cp.Code, cp.DisplayName, cp.DisplayOrder))
	if err != nil {
		return entity.Checkpoint{}, fmt.Errorf("creating checkpoint: %w", err)
	}
	return created, nil
}

// Reorder updates display_order for all checkpoints in a race.
// Uses a temporary offset to avoid unique-constraint conflicts during the swap.
func (r *CheckpointRepo) Reorder(ctx context.Context, raceID int, orderedIDs []int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Shift current orders out of the valid range to avoid transient conflicts.
	if _, err := tx.ExecContext(ctx,
		`UPDATE checkpoints SET display_order = display_order + 10000 WHERE race_id = $1`, raceID); err != nil {
		return fmt.Errorf("shifting display_order: %w", err)
	}

	// Apply final order using unnest to do it in one statement.
	orders := make([]int, len(orderedIDs))
	for i := range orderedIDs {
		orders[i] = i + 1
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE checkpoints AS c SET display_order = v.ord
		 FROM unnest($1::int[], $2::int[]) AS v(id, ord)
		 WHERE c.id = v.id AND c.race_id = $3`,
		pq.Array(orderedIDs), pq.Array(orders), raceID); err != nil {
		return fmt.Errorf("reordering checkpoints: %w", err)
	}

	return tx.Commit()
}
