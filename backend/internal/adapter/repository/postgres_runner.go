package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
)

type RunnerRepo struct {
	db *sql.DB
}

func NewRunnerRepo(db *sql.DB) *RunnerRepo { return &RunnerRepo{db: db} }

var _ portrepo.RunnerRepository = (*RunnerRepo)(nil)

const runnerCols = `id, race_id, bib_number, first_name, last_name, sort_order, status, created_at, updated_at`

func scanRunner(s interface{ Scan(...any) error }) (entity.Runner, error) {
	var r entity.Runner
	err := s.Scan(&r.ID, &r.RaceID, &r.BibNumber, &r.FirstName, &r.LastName,
		&r.SortOrder, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func (r *RunnerRepo) List(ctx context.Context, raceID int) ([]entity.Runner, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+runnerCols+` FROM runners WHERE race_id = $1 ORDER BY sort_order`, raceID)
	if err != nil {
		return nil, fmt.Errorf("listing runners: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var runners []entity.Runner
	for rows.Next() {
		run, err := scanRunner(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning runner: %w", err)
		}
		runners = append(runners, run)
	}
	return runners, rows.Err()
}

func (r *RunnerRepo) Get(ctx context.Context, id int) (entity.Runner, error) {
	run, err := scanRunner(r.db.QueryRowContext(ctx,
		`SELECT `+runnerCols+` FROM runners WHERE id = $1`, id))
	if err != nil {
		return entity.Runner{}, mapNotFound(err)
	}
	return run, nil
}

func (r *RunnerRepo) GetByBibInEvent(ctx context.Context, eventID, bibNumber int) (entity.Runner, error) {
	run, err := scanRunner(r.db.QueryRowContext(ctx,
		`SELECT r.id, r.race_id, r.bib_number, r.first_name, r.last_name,
		        r.sort_order, r.status, r.created_at, r.updated_at
		 FROM runners r
		 JOIN races ra ON ra.id = r.race_id
		 WHERE ra.event_id = $1 AND r.bib_number = $2`,
		eventID, bibNumber))
	if err != nil {
		return entity.Runner{}, mapNotFound(err)
	}
	return run, nil
}

// BulkCreate inserts all runners in a single statement inside a transaction.
func (r *RunnerRepo) BulkCreate(ctx context.Context, runners []entity.Runner) error {
	if len(runners) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(`INSERT INTO runners (race_id, bib_number, first_name, last_name, sort_order, status) VALUES `)
	args := make([]any, 0, len(runners)*6)
	for i, run := range runners {
		if i > 0 {
			sb.WriteString(", ")
		}
		n := i * 6
		fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5, n+6)
		args = append(args, run.RaceID, run.BibNumber, run.FirstName, run.LastName, run.SortOrder, string(run.Status))
	}

	_, err := r.db.ExecContext(ctx, sb.String(), args...)
	if err != nil {
		return fmt.Errorf("bulk creating runners: %w", err)
	}
	return nil
}

func (r *RunnerRepo) UpdateStatus(ctx context.Context, id int, status entity.RunnerStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE runners SET status = $1, updated_at = NOW() WHERE id = $2`, string(status), id)
	return err
}

func (r *RunnerRepo) MaxSortOrder(ctx context.Context, raceID int) (int, error) {
	var max int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sort_order), 0) FROM runners WHERE race_id = $1`, raceID).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("getting max sort order: %w", err)
	}
	return max, nil
}
