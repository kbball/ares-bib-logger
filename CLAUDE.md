# CLAUDE.md — Ares Bib Logger

Ground rules for all development (human and AI) in this repository.

## Stack

- **Backend**: Go, Postgres, `golang-migrate`
- **Frontend**: TypeScript, React, Vite
- **Infrastructure**: Docker / docker-compose
- **AI files**: `.ai/` directory (prompts, specs, decisions, plan)

## Architecture — Hexagonal (Ports & Adapters)

All backend code follows hexagonal architecture. Dependency direction is always inward:

```
adapter → application → domain
```

### Layer rules

| Layer | Path | Rule |
|---|---|---|
| Domain | `backend/internal/domain/` | Pure Go only. Zero framework or infrastructure imports. Contains entities and port interfaces. |
| Application | `backend/internal/application/` | Orchestrates domain via port interfaces. No HTTP, DB, or external service imports. |
| Adapter | `backend/internal/adapter/` | All framework/infra code. Implements or consumes port interfaces. |

The frontend mirrors this structure under `frontend/src/` with `domain/`, `application/`, `adapters/`, and `ui/` layers.

## Configuration — 12-Factor

- All runtime config comes from environment variables. No hardcoded values.
- `.env.example` is the canonical reference for required variables. Commit changes to it.
- `.env` is gitignored. Never commit it.
- Local dev loads env vars via docker-compose `env_file`.

## Database

- Postgres only.
- Migrations managed with `golang-migrate` using plain SQL files (`backend/internal/adapter/repository/migrations/`).
- **Migrations run automatically at application startup** before the HTTP server starts. `main.go` calls `migrate.Up()` on boot.
- New migrations: `make migrate-create NAME=<description>`.

## Logging

- Use `log/slog` (stdlib). No third-party logging libraries.
- JSON handler in production (`ENV=production`), text handler otherwise.
- Log level set via `LOG_LEVEL` env var (`debug`, `info`, `warn`, `error`). Default: `info`.
- Never log sensitive data (passwords, tokens, PII).

## Testing

- Every package must have tests. Target: **>80% coverage**.
- Backend: `testing` stdlib + `testify/assert` + `testify/require` + `testify/mock`. Use `mockery` to generate mocks from port interfaces.
- Frontend: Vitest + React Testing Library + MSW for API mocking.
- Domain and application layers must be unit tested in isolation — no DB, no HTTP.
- Run all tests: `make test`. Run with coverage: `make coverage`.

## Pre-Commit Requirements

Both must pass before any commit:

```
make lint
make fmt
```

- Backend linter: `golangci-lint`
- Frontend linter: ESLint
- Backend formatter: `gofmt`
- Frontend formatter: Prettier

## Plan File

`.ai/PLAN.md` is the source of truth for project progress. Keep it updated:

- Check off completed work as it is done.
- Log architecture decisions with rationale.
- Add new backlog items as they are identified.
