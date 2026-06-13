# Ares Bib Logger — Project Plan

## Work Log

- [x] 2026-06-13 — Defined stack: Go backend, TypeScript/React frontend, Postgres, Docker
- [x] 2026-06-13 — Established hexagonal architecture as the structural pattern
- [x] 2026-06-13 — Created `CLAUDE.md` with ground rules
- [x] 2026-06-13 — Created `.ai/PLAN.md`
- [x] 2026-06-13 — Created `.env.example`
- [x] 2026-06-13 — Created `.gitignore`
- [x] 2026-06-13 — Created `Makefile`
- [x] 2026-06-13 — Created `docker-compose.yml`
- [x] 2026-06-13 — Created `backend/go.mod`
- [x] 2026-06-13 — Scaffolded hexagonal directory structure (backend + frontend)

## Backlog

- [x] 2026-06-13 — Frontend init: package.json, Vite, TypeScript, React, Vitest, ESLint, Prettier

### Backend


### Backend
- [ ] `main.go` — wire env config, run migrations on startup, start HTTP server
- [ ] Domain entities
- [ ] Port interfaces
- [ ] Application services
- [ ] HTTP adapter (routes + handlers)
- [ ] Postgres repository adapter

### Frontend
- [ ] Vite + React + TypeScript scaffold
- [ ] Domain types
- [ ] API adapter (fetch wrappers)
- [ ] UI components + pages

### CI / Quality
- [ ] Pre-commit hook or CI step for `make lint && make fmt`
- [ ] Coverage enforcement in CI

## Architecture Decisions

| Date | Decision | Rationale |
|---|---|---|
| 2026-06-13 | Hexagonal architecture | Keeps domain logic framework-free and independently testable; adapters can be swapped without touching business logic |
| 2026-06-13 | `golang-migrate` for migrations | Plain SQL files (no DSL), strong Go library support, CLI available for Makefile targets |
| 2026-06-13 | Migrations run at startup | Simplifies deployment — no separate migration job or manual step; safe because `golang-migrate` is idempotent |
| 2026-06-13 | `log/slog` for logging | Stdlib in Go 1.21+, structured logging with no added dependency |
| 2026-06-13 | 12-factor env var config | Keeps environment-specific config out of code; supports dev/staging/prod parity |
| 2026-06-13 | Vitest over Jest | Native Vite integration, faster, ESM-first — better fit for the React/Vite frontend |
