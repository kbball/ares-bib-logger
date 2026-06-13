package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
)

type EventRepo struct {
	db *sql.DB
}

func NewEventRepo(db *sql.DB) *EventRepo { return &EventRepo{db: db} }

var _ portrepo.EventRepository = (*EventRepo)(nil)

func (r *EventRepo) List(ctx context.Context) ([]entity.Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, created_at FROM events ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var events []entity.Event
	for rows.Next() {
		var e entity.Event
		if err := rows.Scan(&e.ID, &e.Name, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *EventRepo) Get(ctx context.Context, id int) (entity.Event, error) {
	var e entity.Event
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM events WHERE id = $1`, id).
		Scan(&e.ID, &e.Name, &e.CreatedAt)
	if err != nil {
		return entity.Event{}, mapNotFound(err)
	}
	return e, nil
}

func (r *EventRepo) Create(ctx context.Context, name string) (entity.Event, error) {
	var e entity.Event
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO events (name) VALUES ($1) RETURNING id, name, created_at`, name).
		Scan(&e.ID, &e.Name, &e.CreatedAt)
	if err != nil {
		return entity.Event{}, fmt.Errorf("creating event: %w", err)
	}
	return e, nil
}
