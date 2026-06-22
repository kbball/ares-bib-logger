# Ares Bib Logger

A radio-tent web app for ARES groups supporting ultramarathon events. Replaces the manual Excel + Winlink workflow with automated bib capture via Meshtastic/MQTT, structured runner tracking, and one-click Winlink export.

## Background

Built for the NW-GA ARES team supporting the **GA Death Race (GDR)** and **GA Jewel** ultramarathons in the North Georgia mountains. Each station runs a local instance — there is no shared database between stations. Winlink (radio email) is the only inter-station data channel.

## Features

- **Auto-capture** — subscribes to a local Mosquitto MQTT broker and parses incoming Meshtastic messages; bib numbers extracted and logged with a timestamp automatically
- **Manual entry** — fallback bib entry, DNS, and DNF logging from the UI; `MQTT_ENABLED=false` boots the app in manual-only mode with no MQTT dependency
- **Mesh acks** — after logging, sends a `LOGGED: N` or `DUPLICATE BIB: N` reply back to the Meshtastic mesh via MQTT; the logger node appears as "Auto Logger" on connected devices
- **Winlink export** — generates a ready-to-copy time column (`HH:MM` / `DNS` / `DNF` / `MOVED <raceName>` / blank) plus a pre-built email subject line for the active race checkpoint
- **Winlink import** — paste a column received from another station; same column can be re-imported any number of times (upsert); shows a per-line summary of skipped rows; active checkpoint excluded from source selector to prevent self-import
- **Pace & projected arrival** — once checkpoint distances (miles from start) are configured, displays each runner's current pace and projected arrival time at the next checkpoint; race-stats cards show the earliest expected arrival at the active checkpoint
- **Runner table** — searchable by bib or name; all checkpoint columns with actual logged times; sortable columns; race filter tabs; click any row for a detail panel showing pace, projected arrival, and the full checkpoint log for that runner
- **Race transfer** — move a runner from one GA Jewel race to another mid-event; `MOVED` shown in the original race, runner appended to the new race
- **Event & checkpoint management** — create events and races; define checkpoint order per race (lockable to prevent mid-race shifts); bulk TSV checkpoint import; archive completed events
- **Event export / import** — download a full event config (races, checkpoints, roster) as JSON; import that JSON on another station instead of re-entering everything manually; Winlink-transmittable file size
- **Change runner status** — search by bib, view current status, set to ACTIVE / DNS / DNF / FINISHED without re-logging a bib
- **Context-sensitive help** — `?` button in the app bar opens a per-tab help drawer explaining what each section does
- **Operator guide** — dedicated Guide tab with accordion sections: Before Race Day, On Race Day, Winlink Workflow, Race Transfers, and Tips & Troubleshooting
- **Light / dark mode** — light by default; user-toggleable from the app bar

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

Two tracks depending on your role:

- **Operator** — running the app at a race event; no Go or Node.js required
- **Developer** — working on the codebase

---

### Operator Setup (race-day deployment)

**Prerequisites:** [Docker Desktop](https://www.docker.com/products/docker-desktop/) only.

#### First-time setup

<details>
<summary>macOS / Linux</summary>

```bash
# Download the operator compose file and config
curl -O https://raw.githubusercontent.com/kbball/ares-bib-logger/main/docker-compose.operator.yml
curl -O https://raw.githubusercontent.com/kbball/ares-bib-logger/main/mosquitto.conf

# Create your local config
curl -O https://raw.githubusercontent.com/kbball/ares-bib-logger/main/.env.example
cp .env.example .env
# Open .env and set MQTT_GATEWAY_NODE_ID and MQTT_CHANNEL_NAME; see MQTT Setup below

# Pull the latest image and start everything
docker compose -f docker-compose.operator.yml up -d
```

</details>

<details>
<summary>Windows (PowerShell)</summary>

```powershell
# Download the operator compose file and config
curl.exe -O https://raw.githubusercontent.com/kbball/ares-bib-logger/main/docker-compose.operator.yml
curl.exe -O https://raw.githubusercontent.com/kbball/ares-bib-logger/main/mosquitto.conf

# Create your local config
curl.exe -O https://raw.githubusercontent.com/kbball/ares-bib-logger/main/.env.example
Copy-Item .env.example .env
# Open .env in Notepad and set MQTT_GATEWAY_NODE_ID and MQTT_CHANNEL_NAME; see MQTT Setup below
notepad .env

# Pull the latest image and start everything
docker compose -f docker-compose.operator.yml up -d
```

</details>

<details>
<summary>Windows (Command Prompt)</summary>

```cmd
:: Download the operator compose file and config
curl -O https://raw.githubusercontent.com/kbball/ares-bib-logger/main/docker-compose.operator.yml
curl -O https://raw.githubusercontent.com/kbball/ares-bib-logger/main/mosquitto.conf

:: Create your local config
curl -O https://raw.githubusercontent.com/kbball/ares-bib-logger/main/.env.example
copy .env.example .env
:: Open .env in Notepad and set MQTT_GATEWAY_NODE_ID and MQTT_CHANNEL_NAME; see MQTT Setup below
notepad .env

:: Pull the latest image and start everything
docker compose -f docker-compose.operator.yml up -d
```

</details>

The app is available at `http://localhost:8080`.

#### Update to the latest release

```bash
docker compose -f docker-compose.operator.yml pull
docker compose -f docker-compose.operator.yml up -d
```

#### Stop

```bash
docker compose -f docker-compose.operator.yml down
```

> **Data persistence:** Postgres data survives container restarts via a named Docker volume. To wipe all data (e.g. between events), run `docker compose -f docker-compose.operator.yml down -v`.

---

### Developer Setup

**Prerequisites:**

- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- Go 1.24+
- Node.js 20+
- [golangci-lint](https://golangci-lint.run/) and [golang-migrate CLI](https://github.com/golang-migrate/migrate) (installed via `make install-tools`)

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
├── .github/workflows/
│   ├── ci.yml            # Lint + test on PRs to staging or main
│   ├── staging.yml       # Lint + test + push :staging image on merge to staging
│   └── release.yml       # Manual release: tag + push versioned image + :latest
├── backend/
│   ├── cmd/server/       # Entry point (main.go)
│   └── internal/
│       ├── domain/       # Entities and port interfaces (no framework imports)
│       ├── application/  # Use cases / services
│       └── adapter/      # HTTP handlers, Postgres repos, MQTT client, SSE broker
├── frontend/
│   └── src/
│       ├── domain/       # Core types, interfaces, and pure domain logic (pace computation)
│       ├── adapters/     # API clients, SSE stream hook
│       └── ui/           # React components and pages (six tabs)
├── CLAUDE.md             # Ground rules for AI-assisted development
├── Makefile
├── docker-compose.yml            # Developer: builds backend + frontend locally
├── docker-compose.operator.yml   # Operator: pulls pre-built image from GHCR
└── .env.example
```

## Configuration

All runtime config is via environment variables (12-factor). Copy `.env.example` to `.env` for local development.

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Log level (`debug` / `info` / `warn` / `error`) |
| `TIMEZONE` | `Local` | IANA timezone for Winlink time parsing/formatting (e.g. `America/New_York`). Must match the local timezone of the event venue. |
| `DB_HOST` | `localhost` | Postgres host |
| `DB_PORT` | `5432` | Postgres port |
| `DB_NAME` | `ares_bib_logger` | Database name |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | *(required)* | Database password |
| `DB_SSL_MODE` | `disable` | SSL mode (`disable` / `require` / `verify-full`) |
| `MQTT_ENABLED` | `false` | Set to `true` to enable Meshtastic MQTT integration |
| `MQTT_HOST` | `localhost` | Mosquitto broker host |
| `MQTT_PORT` | `1883` | Mosquitto broker port |
| `MQTT_REGION` | `US` | Meshtastic region prefix used in the MQTT topic (e.g. `US`, `EU`) |
| `MQTT_CHANNEL_NUM` | `2` | Channel number in the MQTT topic path |
| `MQTT_CHANNEL_NAME` | `LongFast` | Channel name in the MQTT topic path — must match the channel name configured on the Meshtastic gateway |
| `MQTT_CHANNEL_INDEX` | `0` | Index (0–7) of the bridged channel in the gateway's channel list; primary channel is `0`, secondary channels are 1+. Check with `meshtastic --info` or the Meshtastic app. |
| `MQTT_GATEWAY_NODE_ID` | *(required when enabled)* | Gateway node ID in hex without `!` (e.g. `a3b4c5d6`). Used to drop echoes of our own messages. |
| `MQTT_NODE_LONG_NAME` | `Auto Logger` | Name the logger node advertises to the mesh (max ~20 chars); shown in Meshtastic app node lists |
| `MQTT_NODE_SHORT_NAME` | `Log` | Short name shown on Meshtastic device screens (max 4 chars) |

**Topic structure:**

```
Subscribe: msh/{MQTT_REGION}/{MQTT_CHANNEL_NUM}/e/{MQTT_CHANNEL_NAME}/#
Publish:   msh/{MQTT_REGION}/{MQTT_CHANNEL_NUM}/e/{MQTT_CHANNEL_NAME}/!ffffffff
```

---

## MQTT / Meshtastic Setup

The MQTT integration uses a local [Mosquitto](https://mosquitto.org/) broker (included in Docker compose) as the bridge between the Meshtastic mesh and the bib logger. A Meshtastic node at the race station acts as the MQTT gateway — it uplinks mesh traffic to the broker and downlinks ack messages back out to the mesh.

### 1. Identify your gateway node ID

In the Meshtastic app, go to **Settings → Device** or look at the node list. The node ID is the 8-character hex value shown as `!a3b4c5d6` — copy it without the `!`.

Set `MQTT_GATEWAY_NODE_ID=a3b4c5d6` in your `.env`.

### 2. Configure MQTT on the gateway node

In the Meshtastic app (or via `meshtastic --configure`), under **Settings → MQTT**:

| Setting | Value |
|---------|-------|
| MQTT enabled | On |
| MQTT server address | IP address of the race station laptop (e.g. `192.168.1.100`) |
| MQTT port | `1883` |
| Root topic | `msh` |
| Encryption disabled | **On** — messages must arrive at the broker as plaintext |
| JSON enabled | Off |
| TLS enabled | Off |

### 3. Configure channel uplink/downlink

In the Meshtastic app, under **Settings → Channels**, for the channel you want to bridge:

| Setting | Value |
|---------|-------|
| Uplink enabled | On |
| Downlink enabled | On |

> The channel name in the Meshtastic app must match `MQTT_CHANNEL_NAME` in your `.env`.
> If you are using a non-primary channel (index > 0), also set `MQTT_CHANNEL_INDEX` accordingly.

### 4. Enable MQTT in the bib logger

In your `.env`:

```env
MQTT_ENABLED=true
MQTT_GATEWAY_NODE_ID=a3b4c5d6   # your gateway's node ID
MQTT_CHANNEL_NAME=LongFast      # must match the channel name in the Meshtastic app
```

Restart the stack (`docker compose -f docker-compose.operator.yml up -d`) and the logger will subscribe to the broker. When the adapter connects it broadcasts a `NODEINFO_APP` packet so the mesh displays the logger as "Auto Logger" / "Log".

### 5. Verify

Send a text message on the configured channel from any Meshtastic device containing a bib number (e.g. `101`). You should see the bib appear in the UI, and a `LOGGED: 101` reply on the mesh from "Auto Logger".

Set `LOG_LEVEL=debug` for verbose MQTT logs during setup.

## CI / CD

Three GitHub Actions workflows implement a `feature → staging → main` pipeline:

| Workflow | Trigger | What it does |
|----------|---------|--------------|
| **CI** | PR to `staging` or `main` | Runs lint + tests; required to pass before merge |
| **Staging** | Push to `staging` | Runs lint + tests, then builds and pushes `ghcr.io/kbball/ares-bib-logger:staging` |
| **Release** | Manual (`workflow_dispatch` from `main`) | Runs lint + tests, creates a `v<major>.<minor>` git tag, builds and pushes versioned image + `:latest` |

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full branch model and release process.

## Development Guidelines

See [CLAUDE.md](CLAUDE.md) for the full set of coding standards. Key points:

- Hexagonal architecture — domain layer has zero framework imports
- All code must have tests; target >90% backend coverage, >80% frontend coverage
- Run `make lint && make fmt` before every commit (pre-commit hook enforces this automatically after `make install`)
- All config via env vars — no hardcoded values

## License

See [LICENSE](LICENSE).
