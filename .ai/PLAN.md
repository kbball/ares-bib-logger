# Ares Bib Logger — Project Plan

## Background

The user is a team lead for an ARES (Amateur Radio Emergency Service) group that supports ultramarathon events in the North Georgia mountains. The group's mission is emergency radio communication and runner tracking on course.

**Events supported:**
- **GA Death Race (GDR)** — single race, single aid station ID per location
- **GA Jewel** — four concurrent races running simultaneously at the same event:
  - 100 Miler
  - 50 Miler
  - 35 Miler
  - 18 Miler

**Current process:**
1. A logger at an aid station captures bib numbers as runners pass through
2. The logger transmits bib numbers via Meshtastic (LoRa mesh radio)
3. Someone at the radio tent receives the message and manually enters bib numbers + time into an Excel spreadsheet (columns A, B, C)
4. GDR = single spreadsheet tab; GA Jewel = four tabs (one per race)
5. 100 Miler aid stations have two IDs: OUT (runners heading away from start) and IN (runners returning inbound toward finish). All other races use a single ID per station.
6. Throughout the race (approximately every 20–30 minutes), the station's time column is copied from the spreadsheet into Winlink (radio email) and transmitted to other stations.
7. Other stations send their Winlink updates back — the receiving station pastes the column into the corresponding aid station column in the spreadsheet to track runners across the whole course.

**Deployment model:** Each station runs this app locally on a laptop. There is no shared database between stations — Winlink is the only mechanism to share information between stations. The app is entirely self-contained per station.

---

## GDR Spreadsheet Analysis

Analyzed: `GDR Runners 2026 NW-GA ARES.xlsx` (~329 runners, single sheet)

### Column layout

| Col | Header | Role |
|-----|--------|------|
| A | Bib # | **Input** — bib number entered as runners are logged during the race (filled in real-time, NOT pre-populated) |
| B | AS # | **Input** — set to current aid station (e.g. "AS #6"), constant for whole sheet |
| C | Time | **Input** — time the runner was seen, or "DNS" / "DNF" |
| D | — | Derived stats (Total, DNF/DNS count, etc.) |
| E | First Name | Roster |
| F | Last Name | Roster |
| G | — | Sequential sort index (1, 2, 3… = roster order, alphabetical by last name) |
| H | Start | Checkpoint times imported from Start station Winlink |
| I–R | AS #1–#10 | Checkpoint times imported from each station's Winlink |

### GDR aid stations
Start, AS #1 through AS #10 (at least 10 checkpoints)

### Winlink export/import format

```
AS #6          ← header (station name — sometimes missing on import)
17:45:00       ← runner sort_order 1
DNS            ← runner sort_order 2
               ← runner sort_order 3 (blank = not yet seen)
DNF            ← runner sort_order 4
…
```
- Values: `HH:MM:SS` | `DNS` | `DNF` | blank
- Import maps by row position → runner sort_order (positional, not bib-keyed)
- Header line may be absent on import — parser must handle both

---

## GA Jewel Spreadsheet Analysis

Analyzed: `Ga Jewel 2024 Spreadsheet INDEX MATCH_v5.xlsx` (4 race tabs + change log)

### Bib ranges (confirmed from roster data)

| Race | Bib Range | Runner Count |
|------|-----------|-------------|
| 100 Miler | 1 – 103 | 103 |
| 50 Miler | 121 – 243 | 123 |
| 35 Miler | ~192 – 358 | 110 |
| 18 Miler | 381 – 1023 | 101 |

Ranges are largely non-overlapping. **The roster import is the authoritative source for bib-to-race assignment.** Bib ranges can serve as a fallback heuristic only.

### Per-tab column layout (same structure across all four tabs)

| Col | Role |
|-----|------|
| A | **Input** — logged bib number |
| B | **Input** — current checkpoint ID (e.g. "StoverRoadOut BoundDepart") |
| C | **Input** — time, DNS, or DNF |
| F | Roster bib number |
| G | Roster runner name (first + last combined in one cell) |
| I–V | Checkpoint times (one column per checkpoint) |

### Checkpoint chains per race

**100 Miler** (13 checkpoints — Out Bound then In Bound):

| ID | Checkpoint |
|----|-----------|
| TradeCenterStart | Trade Center — Start |
| StoverRoadOut BoundDepart | Stover Road — Out Bound |
| SnakeCreekOut BoundArrive | Snake Creek — Out Bound |
| PocketOut BoundDepart | Pocket — Out Bound |
| JohnsMountainOut BoundDepart | Johns Mountain — Out Bound |
| Dry CreekStart LoopsDepart | Dry Creek — Start Loops |
| Dry CreekStart 2nd LoopsDepart | Dry Creek — Start 2nd Loops |
| DryCreekIn BoundDepart | Dry Creek — In Bound |
| JohnsMountainIn BoundDepart | Johns Mountain — In Bound |
| PocketIn BoundDepart | Pocket — In Bound |
| SnakeCreekIn BoundArrive | Snake Creek — In Bound |
| StoverRoadIn BoundDepart | Stover Road — In Bound |
| TradeCenterFinish | Trade Center — Finish |

**50 Miler** (7 checkpoints):
DryCreekStart LoopsStart → DryCreekEnd LoopsDepart → John'sMountainDepart → PocketDepart → SnakeCreekDepart → StoverRoadDepart → TradeCenterFinish

**35 Miler** (6 checkpoints):
DryCreekStart → John'sMountainDepart → PocketDepart → SnakeCreekDepart → StoverRoadDepart → TradeCenterFinish

**18 Miler** (3 checkpoints):
SnakeCreekStart → StoverRoadDepart → TradeCenterFinish

### Key observations
- The same physical locations (Stover Road, Snake Creek, etc.) appear in multiple races with different checkpoint IDs per race
- The 100M checkpoint ID encodes the direction: "StoverRoadOut BoundDepart" vs "StoverRoadIn BoundDepart"
- When a bib arrives at the station, the race is looked up from the roster, and the appropriate checkpoint ID is used automatically
- The 100M direction toggle in the admin panel switches the active checkpoint ID for 100M runners between Out Bound and In Bound variants

---

## Domain Model

```
Event
  ├── name (e.g. "GA Death Race", "GA Jewel")
  └── Races[]
        ├── name (e.g. "100 Miler", "GDR")
        └── Checkpoints[]
              └── code  (e.g. "StoverRoadOut BoundDepart", "AS #6")

Runner
  ├── bib_number
  ├── name  (combined first + last for GA Jewel; separate stored for GDR)
  ├── race (FK → Race)         — can be reassigned during GA Jewel
  ├── sort_order               — row position in roster (drives Winlink export order)
  └── status  (ACTIVE | DNS | DNF | FINISHED | MOVED | UNKNOWN)

CheckpointLog  (one record per runner-per-checkpoint sighting)
  ├── runner (FK → Runner)
  ├── checkpoint (FK → Checkpoint)
  ├── recorded_at  (timestamp)
  ├── source  (MESHTASTIC | MANUAL | WINLINK_IMPORT)
  └── raw_message  (original MQTT payload or pasted text, for audit)

ActiveSession  (one row, updated in place — survives restarts)
  ├── event (FK → Event)
  └── active_checkpoints[]    — one checkpoint per active race at this station
                               (for 100M: switches between Out/In during the race)
```

**Key domain rules:**
- GA Jewel: all four races are active simultaneously. Race is derived from the Runner entity via the roster. The station sets one active checkpoint per race.
- The 100M direction (Out/In) is the only checkpoint that changes mid-race. Switching it updates the 100M entry in active_checkpoints.
- Runners may transfer between races during GA Jewel — updates `Runner.race`.
- Winlink export order = sort_order ascending.
- Winlink import maps by position: row N → runner with sort_order N.
- Roster is the authoritative bib-to-race source. Bib ranges are a fallback heuristic only.

---

## Features

### 1. MQTT / Meshtastic Integration

**Topic structure:** `msh/{region}/{channel_num}/{enc}/{channel_name}/{node_id}`
- Subscribe: `msh/{MQTT_REGION}/{MQTT_CHANNEL_NUM}/e/{MQTT_CHANNEL_NAME}/#`
- Publish (alerts back to mesh): `msh/{MQTT_REGION}/{MQTT_CHANNEL_NUM}/e/{MQTT_CHANNEL_NAME}/!{MQTT_GATEWAY_NODE_ID}`
- `e` in the topic path = gateway-decrypted (plaintext). No encryption — operating under Part 97 amateur radio rules; PSK is `none` on all nodes. Do not add any encryption/decryption logic.

**Inbound message format (JSON `ServiceEnvelope`):**
```json
{
  "from": 2748556758,
  "to": 4294967295,
  "channel": 0,
  "id": 1327955852,
  "rxTime": 1714500000,
  "type": "text",
  "payload": { "text": "101\n202\n303" }
}
```
- Only process messages where `type == "text"`
- `to: 4294967295` (0xffffffff) = broadcast
- `payload.text` contains bib numbers, one per line (`\n` delimited)
- `from` is a uint32 decimal node ID; hex form is `!{hex}` (same value)

**Outbound alert format (published back to gateway topic):**
```json
{
  "channel_id": "LongFast",
  "gateway_id": "!{gateway_node_id}",
  "packet": {
    "from": "{gateway_node_id_as_uint32}",
    "to": 4294967295,
    "decoded": { "portnum": 1, "payload": "DUPLICATE BIB: 101" }
  }
}
```
- Gateway must have `downlink_enabled = true` on the channel to forward to RF
- `from` must be the gateway node's actual node ID

**Processing:**
- Local Mosquitto broker runs in Docker; backend subscribes on startup
- Parse `payload.text`: split on `\n`, strip whitespace, discard non-numeric lines
- Look up each bib → race → active checkpoint → create CheckpointLog
- Store raw JSON payload for audit
- Detect duplicates (same bib, same checkpoint, same session); alert in UI and publish warning back to mesh

### 2. Admin Panel (UI)
Three sections:

**Event & checkpoint configuration**
- Select active event (GDR or GA Jewel)
- For each race: set the active checkpoint ID for this station (dropdown of checkpoints for that race)
- For 100M: checkpoint dropdown handles the Out/In switch — operator just picks the new checkpoint mid-race
- Settings persist in ActiveSession (survive restarts) — safe to update mid-race

**Roster import**
- Race dropdown + large text area for tab-separated paste
- **One-time import per race — enforced at the API level**: the import endpoint returns an error if a roster already exists for that race; the UI reflects this by disabling the import form and showing a locked indicator
- To re-import: the race must be deleted via a separate DELETE endpoint that requires explicit confirmation in the UI; deletes all runners and checkpoint logs for that race
- This prevents accidental roster overwrite mid-race whether the request comes from the UI or directly to the API

**Race/event & checkpoint order configuration** (pre-race setup)
- Create/edit events and races
- Define checkpoint IDs per race and set their display order (this is the column order in the Tabular view and determines which column a Winlink import maps to)
- **Display order locked once the race starts** — enforced at the API level; prevents column shifting mid-race which would break Winlink import mappings
- To change: race must be deleted (with confirmation), wiping all data

### 3. Main UI — Four Tabs

**Tab 1: Data Entry**
- Key race stats per race: total starters, on-course, DNS, DNF, finishers
- Manual bib entry form (source = MANUAL)
- DNS / DNF entry (bib + optional note)
- Recent activity log: last N bibs logged at this station (most recent first)
- Duplicate alert when an incoming bib has already been logged at this station
- Runner race transfer action

**Tab 2: Winlink Import**
- Race selector (for GA Jewel)
- Source checkpoint selector (dropdown of checkpoints in configured display order)
- Large text area: paste received Winlink column
- Submit: parses by row position → sort_order; stores CheckpointLog records
- Import summary: new records added, duplicates skipped

**Tab 3: Winlink Export**
- Race selector (for GA Jewel; GDR auto-selects)
- Auto-populates the active checkpoint ID for this station from ActiveSession — no manual selection needed
- Generates a ready-to-paste column: station header + one time value per runner in sort_order
- Values: `HH:MM:SS` | `DNS` | `DNF` | blank
- **Copy button** to copy the full column to clipboard in one click
- Output refreshes on demand (operator clicks Generate or Copy before each Winlink send)

**Tab 4: Runners (Tabular — view only)**
- Search bar: filter by bib number or runner name (live filter, no page reload)
- Race filter (tab or dropdown for GA Jewel)
- Full runner list in sort_order
- Columns: bib, name, status — then one column per checkpoint in configured display order
- Each cell: time at that checkpoint (our logs or Winlink imports), DNS/DNF, or blank
- Transferred runners appear at bottom of new race; MOVED shown in original race row
- Read-only — no actions on this tab

### 6. Pre-loaded Roster
- Import via paste: user copies three columns (bib, first name, last name) from the spreadsheet and pastes them into a text box in the admin panel; selects the target race from a dropdown; submits
- Paste format: tab-separated rows (what Excel produces on copy), one runner per line — e.g. `123\tJohn\tDoe`
- Insertion order is preserved as sort_order — the order runners appear in the paste becomes the Winlink export order
- Accepts 2 columns (bib + combined name) or 3 columns (bib + first + last); auto-detected from tab count

### 7. Runner Race Transfer
- Admin action: mark a runner as transferred from Race A to Race B
- In Race A: runner status set to MOVED; they remain in the roster at their original sort_order (Winlink export shows blank for them going forward)
- In Race B: runner appended to the BOTTOM of the roster (sort_order = max existing + 1), NOT inserted alphabetically
- This preserves the positional integrity of existing Winlink exports for Race A while correctly placing the transferred runner at the end of Race B's export column

---

## Open Questions

| # | Question | Status |
|---|----------|--------|
| 1 | GA Jewel bib ranges | **Resolved** — largely non-overlapping; roster is authoritative |
| 2 | GA Jewel tab/checkpoint structure | **Resolved** — see analysis above |
| 3 | Meshtastic message format | **Resolved** — JSON ServiceEnvelope; `type=text`, bibs in `payload.text` one per `\n` |
| 4 | MQTT topic | **Resolved** — `msh/{region}/{channel_num}/e/{channel_name}/#`; all parts configurable via env |
| 5 | Roster import format | **Resolved** — TSV paste into text box in admin panel; 2 or 3 columns auto-detected |

---

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
- [x] 2026-06-13 — Frontend init: package.json, Vite, TypeScript, React, Vitest, ESLint, Prettier
- [x] 2026-06-13 — Captured project background, domain model, features, and open questions in plan
- [x] 2026-06-13 — Analyzed GDR spreadsheet: Winlink format, column layout, roster structure
- [x] 2026-06-13 — Analyzed GA Jewel spreadsheet: 4 races, bib ranges, checkpoint chains, Out/In structure

## Backlog

### Pending answers / research
- [ ] Confirm Meshtastic message format and MQTT topic
- [ ] Confirm roster import UX (xlsx upload vs CSV)

### Infrastructure
- [ ] Add Mosquitto MQTT broker to `docker-compose.yml`
- [ ] Add MQTT env vars to `.env.example` (MQTT_HOST, MQTT_PORT, MQTT_TOPIC)

### Backend — Foundation
- [ ] `main.go` — env config, run migrations at startup, start HTTP server
- [ ] `config` package — load all env vars into a typed struct
- [ ] DB schema migrations: Event, Race, Checkpoint, Runner, CheckpointLog, ActiveSession

### Backend — Core Domain
- [ ] Domain entities (Event, Race, Checkpoint, Runner, CheckpointLog, ActiveSession)
- [ ] Port interfaces (repository + service layers)
- [ ] Application services (CheckpointLogService, RunnerService, SessionService, WinlinkService)

### Backend — Adapters
- [ ] Postgres repository implementations
- [ ] HTTP handlers (REST API)
- [ ] MQTT adapter (subscribe, parse bibs, dispatch to CheckpointLogService)
- [ ] Winlink export formatter
- [ ] Winlink import parser (positional, optional header)
- [ ] Roster importer (xlsx / CSV → Runner rows)

### Frontend
- [ ] Admin panel: event config, checkpoint display order (locked after race start), roster import (locked after first import), active checkpoint selection per race
- [ ] Tab 1 — Data entry: race stats, manual bib entry, DNS/DNF, recent log, duplicate alert, race transfer
- [ ] Tab 2 — Winlink import: race + checkpoint selector, paste area, import summary
- [ ] Tab 3 — Winlink export: race selector, auto-pulls active AS, generates column, copy button
- [ ] Tab 4 — Runners tabular (view only): bib/name search, checkpoint columns in display order

### CI / Quality
- [ ] Pre-commit hook or CI step for `make lint && make fmt`
- [ ] Coverage enforcement

---

## Architecture Decisions

| Date | Decision | Rationale |
|---|---|---|
| 2026-06-13 | Hexagonal architecture | Keeps domain logic framework-free and independently testable |
| 2026-06-13 | `golang-migrate` for migrations | Plain SQL, strong Go library support, Makefile targets |
| 2026-06-13 | Migrations run at startup | No separate migration job; `golang-migrate` is idempotent |
| 2026-06-13 | `log/slog` for logging | Stdlib Go 1.21+, structured, no added dependency |
| 2026-06-13 | 12-factor env var config | Environment-specific config out of code |
| 2026-06-13 | Vitest over Jest | Vite-native, faster, ESM-first |
| 2026-06-13 | MQTT as a driven adapter | Input source only; domain stays clean if transport changes |
| 2026-06-13 | ActiveSession stored in DB | Survives restarts; critical at a race event |
| 2026-06-13 | Local Mosquitto broker in Docker | Self-contained; no external dependency |
| 2026-06-13 | Single-station deployment | Winlink is the only inter-station channel; app scope is one station |
| 2026-06-13 | Race derived from runner roster | For GA Jewel, race identity lives on Runner not the station config |
| 2026-06-13 | sort_order drives Winlink I/O | Export in roster row order; import maps positionally back to sort_order |
| 2026-06-13 | Winlink import as first-class feature | Closes the loop on DNS/DNF and cross-station checkpoint data |
| 2026-06-13 | ActiveSession holds one checkpoint per race | GA Jewel has 4 concurrent races; each needs its own active checkpoint at the station |
| 2026-06-13 | Roster is authoritative for bib-to-race | Bib ranges not perfectly clean; roster import is required before race starts |
| 2026-06-13 | Roster import locked at API level after first import | UI lock is cosmetic; API must enforce the rule to prevent mid-race overwrite regardless of how the request arrives |
| 2026-06-13 | Checkpoint display order locked at API level after race starts | Column order shift mid-race would break positional Winlink import mappings |
| 2026-06-13 | Runner MOVED status + append-to-bottom in new race | Preserves existing sort_order in the original race (no column shifting on export); transferred runner goes to end of new race |
| 2026-06-13 | Roster import via paste (TSV) not file upload | Lowest friction at race-day — operator already has the spreadsheet open, just copies 3 cols and pastes |
