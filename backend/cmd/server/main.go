package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"

	httphandler "github.com/kevinball/ares-bib-logger/backend/internal/adapter/http/handler"
	mqttadapter "github.com/kevinball/ares-bib-logger/backend/internal/adapter/mqtt"
	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/repository"
	sseadapter "github.com/kevinball/ares-bib-logger/backend/internal/adapter/sse"
	"github.com/kevinball/ares-bib-logger/backend/internal/application/service"
	"github.com/kevinball/ares-bib-logger/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	setupLogger(cfg)

	db, err := connectDB(cfg.DB)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close database connection", "error", err)
		}
	}()

	if err := runMigrations(db); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Repositories
	eventRepo := repository.NewEventRepo(db)
	raceRepo := repository.NewRaceRepo(db)
	checkpointRepo := repository.NewCheckpointRepo(db)
	runnerRepo := repository.NewRunnerRepo(db)
	checkpointLogRepo := repository.NewCheckpointLogRepo(db)
	sessionRepo := repository.NewActiveSessionRepo(db)

	// SSE broker — shared by HTTP handler and MQTT adapter
	broker := sseadapter.NewBroker()

	// Application services
	checkpointLogSvc := service.NewCheckpointLogService(runnerRepo, checkpointLogRepo, sessionRepo)

	if cfg.MQTT.Enabled {
		mqttA, err := mqttadapter.New(cfg.MQTT, checkpointLogSvc, broker)
		if err != nil {
			slog.Error("failed to start MQTT adapter", "error", err)
			os.Exit(1)
		}
		defer mqttA.Stop()
		slog.Info("MQTT adapter started",
			"broker", fmt.Sprintf("%s:%d", cfg.MQTT.Host, cfg.MQTT.Port),
			"subscribe", cfg.MQTT.SubscribeTopic(),
		)
	} else {
		slog.Info("MQTT disabled — running in manual-entry mode")
	}

	h := httphandler.New(
		service.NewEventService(eventRepo),
		service.NewRaceService(raceRepo),
		service.NewCheckpointService(checkpointRepo, raceRepo),
		service.NewRunnerService(runnerRepo, raceRepo),
		checkpointLogSvc,
		service.NewSessionService(sessionRepo),
		service.NewWinlinkService(runnerRepo, checkpointRepo, checkpointLogRepo, sessionRepo),
		broker,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.Handle("GET /api/stream", broker)
	h.Register(mux)
	mux.Handle("/", serveSPA("frontend/dist"))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server started", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}

func setupLogger(cfg *config.Config) {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}

func connectDB(cfg config.DBConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening connection: %w", err)
	}

	for attempt := range 10 {
		if err = db.Ping(); err == nil {
			slog.Info("database connected")
			return db, nil
		}
		slog.Info("waiting for database", "attempt", attempt+1, "error", err)
		time.Sleep(time.Second)
	}

	return nil, fmt.Errorf("database not ready after 10 attempts: %w", err)
}

func runMigrations(db *sql.DB) error {
	src, err := iofs.New(repository.MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("creating migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("running migrations: %w", err)
	}

	slog.Info("migrations applied")
	return nil
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// serveSPA serves a directory of static files and falls back to index.html
// for any path that doesn't match a real file (SPA client-side routing).
func serveSPA(dir string) http.Handler {
	root := os.DirFS(dir)
	fileServer := http.FileServerFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fs.Stat(root, r.URL.Path[1:])
		if err != nil {
			// Not a real file — serve index.html so the SPA router handles it.
			http.ServeFileFS(w, r, root, "index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
