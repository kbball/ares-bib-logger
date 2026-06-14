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
- Values: `HH:MM` | `DNS` | `DNF` | blank
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

**Fallback / manual-entry mode:**
- Controlled by `MQTT_ENABLED` env var (default `true`)
- When `false`: MQTT adapter does not start; app runs fully on manual entry via UI
- No degradation to other features — all UI tabs, Winlink import/export, and tabular view work normally
- Useful when Meshtastic infrastructure is unavailable or being tested without a gateway

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
- Import summary: Created / Updated / Skipped counts; table of skipped details with position, bib, and reason (blank line, no runner at position, duplicate, parse error)

**Tab 3: Winlink Export**
- Race selector (for GA Jewel; GDR auto-selects)
- Auto-populates the active checkpoint ID for this station from ActiveSession — no manual selection needed
- Generates a ready-to-paste column: station header + one time value per runner in sort_order
- Values: `HH:MM` | `DNS` | `DNF` | `MOVED <raceName>` | blank
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
- [x] 2026-06-13 — Created `docker-compose.yml` (includes Mosquitto MQTT broker)
- [x] 2026-06-13 — Created `backend/go.mod`
- [x] 2026-06-13 — Scaffolded hexagonal directory structure (backend + frontend)
- [x] 2026-06-13 — Frontend init: package.json, Vite, TypeScript, React, Vitest, ESLint, Prettier
- [x] 2026-06-13 — Captured project background, domain model, features, and open questions in plan
- [x] 2026-06-13 — Analyzed GDR spreadsheet: Winlink format, column layout, roster structure
- [x] 2026-06-13 — Analyzed GA Jewel spreadsheet: 4 races, bib ranges, checkpoint chains, Out/In structure
- [x] 2026-06-13 — Backend foundation: config package, `main.go`, DB migrations (Event, Race, Checkpoint, Runner, CheckpointLog, ActiveSession), HTTP server
- [x] 2026-06-13 — Backend complete: all domain entities, port interfaces, application services, Postgres repos, HTTP handlers, MQTT adapter, Winlink import/export, roster importer
- [x] 2026-06-13 — Frontend complete: Material UI, all five tabs (Data Entry, Winlink Import, Winlink Export, Runners, Admin), SSE real-time updates, Dockerfile
- [x] 2026-06-13 — MUI dark theme: custom color palette, Inter font (offline via `@fontsource/inter`), component overrides throughout
- [x] 2026-06-13 — Logo: background-removed PNG added to AppBar
- [x] 2026-06-13 — Bug fixes (round 1): checkpoint duplicate key, roster TSV import, transferRunner API, reorderCheckpoints field name, frontend–API alignment
- [x] 2026-06-13 — Enhancements (round 1): checkpoint delete, up/down reorder arrows, HTTP logging middleware, runner table grid lines + sortable columns, race stat tooltips
- [x] 2026-06-13 — Archive event: migration + full stack (repo → service → handler → frontend), removes event from dropdown while preserving data
- [x] 2026-06-13 — Lock checkpoint order: `PUT /api/races/{id}/lock-order`, hides reorder arrows and add-checkpoint form in Admin once locked
- [x] 2026-06-13 — DNS/DNF Winlink import: now also creates a CheckpointLog entry so runners appear in checkpoint columns on Runners tab
- [x] 2026-06-13 — `GET /api/races/{raceID}/logs` endpoint + frontend wiring: Runners tab now shows actual logged times (HH:MM) and DNS/DNF per checkpoint cell
- [x] 2026-06-13 — Data Entry tab: refreshes runner stats after every log/DNS/DNF/transfer; shows checkpoint name+code instead of ID; disables Log/Submit when no active CP set
- [x] 2026-06-13 — Winlink Import tab: import summary now shows Created count alongside Updated/Skipped
- [x] 2026-06-13 — Runners tab: default sort changed to Bib ascending; MOVED chip changed to orange (warning); checkpoint columns populated from log data
- [x] 2026-06-13 — Light/dark mode: theme factory (`createAppTheme(mode)`), sun/moon toggle icon in AppBar top-right, dark is default
- [x] 2026-06-13 — Winlink export: MOVED runners now emit `MOVED <raceName>` instead of a blank line; WinlinkService gains races repo to resolve the target race across the event
- [x] 2026-06-13 — Winlink import: added `SkippedDetails` to result (position, bib, reason: blank/no_runner/duplicate/parse_error); Import tab displays a details table when skips occur
- [x] 2026-06-13 — Test fixes: added missing mock stubs (Archive, LockOrder, Update/Delete on checkpoints, ListByRace on log service) across service and handler test packages; fixed reorder test field name, roster test format/status, export format timezone
- [x] 2026-06-13 — Pace / Projected Arrival: migration 000003 adds nullable `distance_from_start` to checkpoints; field threaded through entity → repo → service → handler; Admin UI adds Dist (mi) field to checkpoint create/edit; `domain/pace.ts` computes pace from last two logged CPs with distances; Runners tab shows Pace (MM:SS /mi) and Proj. Next (HH:MM) columns when ≥2 CPs have distances; Data Entry race cards show "Next expected: HH:MM" at the active checkpoint
- [x] 2026-06-14 — **Bug fix (Winlink blank-line positional shift)**: `looksLikeTimeOrStatus` failed to recognize single-digit-hour times (`"7:35"`, len=4) because the check required `len(s) >= 5`; the first data row was misidentified as a checkpoint header and skipped, shifting every subsequent runner one position off; fixed by also matching `H:MM` / `H:MM:SS` patterns via `s[1] == ':'`
- [x] 2026-06-14 — Winlink import upsert: changed from skip-on-duplicate to upsert — same column can be re-imported any number of times and new data overwrites existing; added `Upsert` to `CheckpointLogRepository` using `INSERT … ON CONFLICT (runner_id, checkpoint_id) DO UPDATE`; `xmax=0` detects insert vs overwrite for `Created`/`Updated` result counts; manual and MQTT logging remain dedup-only (unaffected); removed "duplicate" skip reason from frontend label map
- [x] 2026-06-14 — Default theme changed from dark to light
- [x] 2026-06-14 — Admin: "Change Runner Status" section — select race, enter bib, click Search to find runner; shows name + current status chip → new-status dropdown (ACTIVE / DNS / DNF / FINISHED) + Set button; calls existing `POST /api/log/status`
- [x] 2026-06-14 — Runners tab: clicking any row opens a runner detail modal — shows bib, race, status chip, current pace, projected arrival at the active checkpoint (using display name), and full checkpoint log table for that runner's race
- [x] 2026-06-14 — Frontend test suite: Vitest + React Testing Library + MSW; 163 tests across all layers (domain/pace, API adapters, App, and all five tab components); useStream SSE callbacks covered via mock-capture pattern; 89% branch coverage (461/517), meets the 80% frontend threshold; coverage thresholds enforced in `vite.config.ts`; coverage targets split in `CLAUDE.md` (backend >90%, frontend >80%)
- [x] 2026-06-14 — Backend test suite: full coverage pass targeting >90% per-package; config test isolation via `clearEnv(t)` + `t.Setenv` to eliminate shell env bleed; domain entity constants tests (RunnerStatus, LogSource string values locked against rename); domain sentinel error tests (distinct, wrappable, correct messages); repository layer rewritten with `go-sqlmock` (v1.5.2) to run without a live DB — 70+ sqlmock tests covering all repos (event, race, checkpoint, runner, checkpoint_log, active_session) including transactions, nullable columns, and upsert; HTTP handler tests expanded to 97.3% coverage (updateCheckpoint, deleteCheckpoint, archiveEvent, listCheckpointLogs, lockRaceOrder, LoggingMiddleware/WriteHeader, parseTSVRoster 2-column paths, publishSession error branch); application service tests expanded to 95.9% coverage (CheckpointService.Create auto-order + list-error, Update/Delete all error paths, EventService.Archive, RaceService.LockOrder, CheckpointLogService.ListByRace)

## Backlog

### ~~Winlink Import — Blank Line Positional Investigation~~ ✅ Resolved 2026-06-14
Root cause identified and fixed: single-digit-hour times (e.g. `7:35`) were 4 chars and failed the `len(s) >= 5` guard in `looksLikeTimeOrStatus`, causing the first data row to be skipped as a phantom header.

### Frontend — Responsive Layout
- [ ] Data Entry tab: cards stack vertically and use full-width inputs on small screens; target tablet (768px+) as primary field-use form factor

### ~~Frontend + API — Pace / Projected Arrival~~ ✅ Completed 2026-06-13

### ~~Frontend Testing~~ ✅ Completed 2026-06-14
Vitest + React Testing Library + MSW; 163 tests, 89% branch coverage (461/517), thresholds enforced in `vite.config.ts`.

### ~~Backend Testing~~ ✅ Completed 2026-06-14
All packages at >90% coverage: handler 97.3%, service 95.9%, repository 97.3%, config 93.6%, mqtt 92.7%, sse 94.4%. `make coverage` Makefile target was already wired.

### UI — Bulk Checkpoint Import (Priority: High)
- [ ] Admin tab: accept TSV paste (`code\tname\tdist_from_start`) to create multiple checkpoints at once for a race; mirrors the roster import UX pattern

### UI / API — Winlink Export Email Subject Line (Priority: High)
- [ ] Generate a ready-to-copy email subject above the column text area; format: `<CP Name> <Race Name> <HH:MM 24-hr> update`; own text field + copy button

### README — Developer vs. Deployment Setup Docs (Priority: High)
- [ ] Split setup instructions into two tracks: (1) developer / dev-mode, (2) operator deployment (no dev tools required — pull pre-built image from container registry and run)
- [ ] Publish Docker image to a container registry (e.g. GitHub Container Registry) via CI/CD; provide `docker-compose.yml` operators can use without building
- [ ] Add GitHub Actions CI/CD pipeline for image build and push on merge to main

### UI — Context-Sensitive Help Panel (Priority: Medium)
- [ ] Persistent help icon (question mark) in the lower-right corner of every tab; clicking opens a slide-in side panel explaining what the current screen does and how to use it

### UI — Tooltip on All Action Buttons and Icons (Priority: Medium)
- [ ] Add `Tooltip` wrappers to every button and icon that performs an action; keep tooltip text short and action-oriented

### UI — Training / Onboarding Section (Priority: Low)
- [ ] Dedicated training section (tab or modal) that walks a new operator through how to use the application end-to-end before race day

### UI / API — Event Export / Import (Priority: Low)
- [ ] Export a complete event configuration (event, races, checkpoints, roster) to a compact JSON or YAML file
- [ ] Import that file to recreate the event configuration on another station; enables one person to configure once and share via Winlink
- [ ] File size must be minimal to be Winlink-transmittable; omit logs, only include structural config

### .env.example — MQTT Disabled by Default (Priority: Low)
- [ ] Change `MQTT_ENABLED` default to `false` in `.env.example`; operators explicitly opt in when Meshtastic infrastructure is present

### ~~UI — Roster Import: Support `bib,fullName` TSV Format~~ ✅ Completed 2026-06-13
`parseTSVRoster` auto-detects 2-column (`bib\tFull Name`) vs 3-column (`bib\tfirst\tlast`); first-space split derives first/last for 2-column case.

### CI / Quality
- [ ] Pre-commit hook or CI step: `make lint && make fmt` must pass before commit

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
| 2026-06-13 | `MQTT_ENABLED` flag for fallback mode | MQTT is optional; app runs fully in manual-entry mode when disabled — degrades gracefully, doesn't crash |
| 2026-06-13 | All state is DB-persisted; no in-memory-only state | Container restarts must be transparent — event config, roster, checkpoints, and ActiveSession all live in Postgres; the app loads from DB on boot, not from memory |
| 2026-06-13 | Material UI (`@mui/material`) as frontend component library | Pre-built accessible components (tabs, tables, forms, dialogs) match the app's UI well and avoid building layout primitives from scratch |
| 2026-06-13 | `@fontsource/inter` for Inter typeface | App runs off-grid with no internet; CDN fonts are forbidden; npm-bundled font files are the only safe option |
| 2026-06-13 | Dark mode default, user-toggleable light mode | Field use is often in low-light or tent environments — dark default reduces eye strain; light mode available for daylight use |
| 2026-06-13 | Theme as `createAppTheme(mode)` factory | Single source of truth for both themes; `App.tsx` holds the `colorMode` state and passes it to `ThemeProvider` via `useMemo` |
