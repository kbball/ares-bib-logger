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

const eventCols = `id, name, archived, winlink_blank_line_after_header, winlink_reminder_minutes, created_at`

func scanEvent(s interface{ Scan(...any) error }) (entity.Event, error) {
	var e entity.Event
	err := s.Scan(&e.ID, &e.Name, &e.Archived, &e.WinlinkBlankLineAfterHeader, &e.WinlinkReminderMinutes, &e.CreatedAt)
	return e, err
}

func (r *EventRepo) List(ctx context.Context) ([]entity.Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+eventCols+` FROM events WHERE NOT archived ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var events []entity.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *EventRepo) Get(ctx context.Context, id int) (entity.Event, error) {
	e, err := scanEvent(r.db.QueryRowContext(ctx,
		`SELECT `+eventCols+` FROM events WHERE id = $1`, id))
	if err != nil {
		return entity.Event{}, mapNotFound(err)
	}
	return e, nil
}

func (r *EventRepo) Create(ctx context.Context, name string) (entity.Event, error) {
	e, err := scanEvent(r.db.QueryRowContext(ctx,
		`INSERT INTO events (name) VALUES ($1) RETURNING `+eventCols, name))
	if err != nil {
		return entity.Event{}, fmt.Errorf("creating event: %w", err)
	}
	return e, nil
}

func (r *EventRepo) Archive(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE events SET archived = true WHERE id = $1`, id)
	return err
}

func (r *EventRepo) SetWinlinkBlankLineAfterHeader(ctx context.Context, id int, enabled bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE events SET winlink_blank_line_after_header = $1 WHERE id = $2`, enabled, id)
	return err
}

func (r *EventRepo) SetWinlinkReminderMinutes(ctx context.Context, id, minutes int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE events SET winlink_reminder_minutes = $1 WHERE id = $2`, minutes, id)
	return err
}
