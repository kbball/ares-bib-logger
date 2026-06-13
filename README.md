# Ares Bib Logger

A radio-tent web app for ARES groups supporting ultramarathon events. Replaces the manual Excel + Winlink workflow with automated bib capture via Meshtastic/MQTT, structured runner tracking, and one-click Winlink export.

## Background

Built for the NW-GA ARES team supporting the **GA Death Race (GDR)** and **GA Jewel** ultramarathons in the North Georgia mountains. Each station runs a local instance — there is no shared database between stations. Winlink (radio email) is the only inter-station data channel.

## Features

- **Auto-capture** — subscribes to a local Mosquitto MQTT broker and parses incoming Meshtastic messages; bib numbers extracted automatically and logged with a timestamp
- **Manual entry** — fallback bib entry, DNS, and DNF logging from the UI
- **Duplicate detection** — alerts on repeated bibs and optionally rebroadcasts a warning via MQTT
- **Winlink export** — generates a ready-to-copy time column for the active race, every 20–30 minutes throughout the event
- **Winlink import** — paste a column received from another station to capture their checkpoint data and DNS/DNF updates
- **Runner table** — searchable by bib or name; shows all checkpoints in configured order
- **Race transfer** — move a runner from one GA Jewel race to another mid-event

## Events Supported

| Event | Races | Notes |
|-------|-------|-------|
| GA Death Race (GDR) | Single race | ~329 runners, 10 aid stations |
| GA Jewel | 100M / 50M / 35M / 18M | All four races active simultaneously; 100M has Out Bound / In Bound checkpoint phases |

## Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.24, `log/slog`, `golang-migrate` |
| Frontend | TypeScript, React 19, Vite |
| Database | PostgreSQL 16 |
| Messaging | Mosquitto MQTT (local Docker service) |
| Container | Docker / docker-compose |

Architecture follows the **hexagonal (ports & adapters)** pattern — domain logic has zero framework dependencies and is independently testable.

## Getting Started

### Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- Go 1.24+
- Node.js 20+
- [golangci-lint](https://golangci-lint.run/) and [golang-migrate CLI](https://github.com/golang-migrate/migrate) (installed via `make install-tools`)

### Setup

```bash
# Clone the repo
git clone git@github.com:kbball/ares-bib-logger.git
cd ares-bib-logger

# Create local env file
cp .env.example .env

# Install Go tools and frontend dependencies
make install

# Start Postgres and MQTT broker, then run the app
make dev
```

The Go backend starts on `http://localhost:8080`. The Vite frontend dev server starts on `http://localhost:5173` and proxies `/api` calls to the backend.

### Common commands

```bash
make dev            # Start Postgres + MQTT (Docker) then backend + frontend natively
make db-up          # Start only Postgres and MQTT broker
make db-down        # Stop Docker services

make test           # Run all tests (backend + frontend)
make coverage       # Run tests with coverage reports
make lint           # Lint backend and frontend
make fmt            # Format backend and frontend

make migrate-up                      # Apply pending migrations
make migrate-down                    # Roll back last migration
make migrate-create NAME=add_runners # Scaffold a new migration
make migrate-status                  # Show current migration version
```

## Project Structure

```
ares-bib-logger/
├── .ai/                  # AI context — plan, specs, decisions
│   └── PLAN.md           # Living project plan (work log + backlog + arch decisions)
├── backend/
│   ├── cmd/server/       # Entry point
│   └── internal/
│       ├── domain/       # Entities and port interfaces (no framework imports)
│       ├── application/  # Use cases / services
│       └── adapter/      # HTTP handlers, Postgres repos, MQTT client
├── frontend/
│   └── src/
│       ├── domain/       # Core types and interfaces
│       ├── application/  # Custom hooks / use cases
│       ├── adapters/     # API clients, storage
│       └── ui/           # React components and pages
├── CLAUDE.md             # Ground rules for AI-assisted development
├── Makefile
├── docker-compose.yml
└── .env.example
```

## Configuration

All runtime config is via environment variables (12-factor). Copy `.env.example` to `.env` for local development.

| Variable | Default | Description |
|----------|---------|-------------|
| `ENV` | `development` | Environment (`development` / `production`) |
| `SERVER_PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Log level (`debug` / `info` / `warn` / `error`) |
| `DB_HOST` | `localhost` | Postgres host |
| `DB_PORT` | `5432` | Postgres port |
| `DB_NAME` | `ares_bib_logger` | Database name |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | `postgres` | Database password |
| `DB_SSL_MODE` | `disable` | SSL mode (`disable` / `require` / `verify-full`) |

MQTT configuration will be added once Meshtastic message format is confirmed.

## Development Guidelines

See [CLAUDE.md](CLAUDE.md) for the full set of coding standards. Key points:

- Hexagonal architecture — domain layer has zero framework imports
- All code must have tests; target >80% coverage
- Run `make lint && make fmt` before every commit
- All config via env vars — no hardcoded values

## License

See [LICENSE](LICENSE).
