package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/repository"
)

var integrationDB *sql.DB

func TestMain(m *testing.M) {
	dsn := os.Getenv("DB_TEST_DSN")
	if dsn != "" {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			log.Fatalf("opening test DB: %v", err)
		}
		if err := db.Ping(); err != nil {
			log.Fatalf("pinging test DB: %v", err)
		}
		if err := runMigrations(db); err != nil {
			log.Fatalf("running migrations: %v", err)
		}
		integrationDB = db
		defer func() { _ = db.Close() }()
	}
	os.Exit(m.Run())
}

func requireDB(t *testing.T) *sql.DB {
	t.Helper()
	if integrationDB == nil {
		t.Skip("DB_TEST_DSN not set; skipping integration test")
	}
	truncateAll(t)
	return integrationDB
}

func truncateAll(t *testing.T) {
	t.Helper()
	_, err := integrationDB.Exec(`
		TRUNCATE active_session_checkpoints, checkpoint_logs, runners, checkpoints, races, events CASCADE;
		INSERT INTO active_session (id) VALUES (1)
		  ON CONFLICT (id) DO UPDATE SET event_id = NULL, updated_at = NOW();
	`)
	if err != nil {
		t.Fatalf("truncating tables: %v", err)
	}
}

// seedEvent creates a test event and returns its ID.
func seedEvent(t *testing.T, db *sql.DB) int {
	t.Helper()
	event, err := repository.NewEventRepo(db).Create(context.Background(), "Test Event")
	require.NoError(t, err)
	return event.ID
}

// seedRace creates a test race within an event and returns its ID.
func seedRace(t *testing.T, db *sql.DB, eventID int) int {
	t.Helper()
	race, err := repository.NewRaceRepo(db).Create(context.Background(), eventID, "Test Race")
	require.NoError(t, err)
	return race.ID
}

func runMigrations(db *sql.DB) error {
	src, err := iofs.New(repository.MigrationsFS, "migrations")
	if err != nil {
		return err
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
