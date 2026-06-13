package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
)

type RaceRepo struct {
	db *sql.DB
}

func NewRaceRepo(db *sql.DB) *RaceRepo { return &RaceRepo{db: db} }

var _ portrepo.RaceRepository = (*RaceRepo)(nil)

const raceCols = `id, event_id, name, roster_locked, order_locked, created_at`

func scanRace(s interface{ Scan(...any) error }) (entity.Race, error) {
	var r entity.Race
	err := s.Scan(&r.ID, &r.EventID, &r.Name, &r.RosterLocked, &r.OrderLocked, &r.CreatedAt)
	return r, err
}

func (r *RaceRepo) List(ctx context.Context, eventID int) ([]entity.Race, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+raceCols+` FROM races WHERE event_id = $1 ORDER BY created_at`, eventID)
	if err != nil {
		return nil, fmt.Errorf("listing races: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var races []entity.Race
	for rows.Next() {
		race, err := scanRace(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning race: %w", err)
		}
		races = append(races, race)
	}
	return races, rows.Err()
}

func (r *RaceRepo) Get(ctx context.Context, id int) (entity.Race, error) {
	race, err := scanRace(r.db.QueryRowContext(ctx,
		`SELECT `+raceCols+` FROM races WHERE id = $1`, id))
	if err != nil {
		return entity.Race{}, mapNotFound(err)
	}
	return race, nil
}

func (r *RaceRepo) Create(ctx context.Context, eventID int, name string) (entity.Race, error) {
	race, err := scanRace(r.db.QueryRowContext(ctx,
		`INSERT INTO races (event_id, name) VALUES ($1, $2)
		 RETURNING `+raceCols, eventID, name))
	if err != nil {
		return entity.Race{}, fmt.Errorf("creating race: %w", err)
	}
	return race, nil
}

func (r *RaceRepo) LockRoster(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE races SET roster_locked = true WHERE id = $1`, id)
	return err
}

func (r *RaceRepo) LockOrder(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE races SET order_locked = true WHERE id = $1`, id)
	return err
}

func (r *RaceRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM races WHERE id = $1`, id)
	return err
}
