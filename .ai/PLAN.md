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
- [x] 2026-06-14 — Developer vs. Deployment docs: README split into Operator (Docker-only) and Developer tracks; `docker-compose.operator.yml` added for operators pulling pre-built image from GHCR; `.github/workflows/ci.yml` added — runs `test` and `lint` jobs on PRs, then builds and pushes `ghcr.io/kbball/ares-bib-logger:latest` + `sha-*` tag to GHCR on merge to main using GitHub Actions cache for layer reuse
- [x] 2026-06-14 — Winlink Export email subject line: subject field (`<CP DisplayName> <Race Name> <HH:MM> update`) appears above the column textarea after Generate; read-only text field + Copy Subject button; Copy Subject recomputes the current time at click time; 4 new tests (subject visible, correct content, clipboard copy, Copied! feedback)
- [x] 2026-06-14 — Tooltips on all action buttons and icons: MUI `Tooltip` added to every action button (Generate, Copy Subject, Copy to Clipboard, Import, Log, Submit, Transfer) and all AdminTab icon buttons (Archive, Lock Order, Delete Race, Save, Cancel, Move Up/Down, Edit CP, Delete CP); `describeChild` used on text buttons so tooltip uses `aria-describedby` rather than overriding the button accessible name with `aria-label`; test queries migrated from `getByTitle` to `getByRole('button', { name: ... })`; all 166 tests pass
- [x] 2026-06-14 — Bug fix (Data Entry cards not refreshing): `useStream` captured handlers once at mount with empty deps so `races` was always `[]` inside the SSE callback; fixed with a `handlersRef` updated each render so the EventSource always invokes the latest closure; also added `loadLogs` to `submitBib` and `submitStatus` for direct refresh
- [x] 2026-06-14 — Bug fix (Winlink timezone): Docker container timezone is UTC; `parseTimeOfDay` used `time.Local` (= UTC), storing imported times as UTC and the export formatting `RecordedAt.Local()` also produced UTC; fixed by adding `TIMEZONE` env var (IANA name, e.g. `America/New_York`) to config, threading `*time.Location` into `WinlinkService`, and using it in both import parsing and export formatting
- [x] 2026-06-14 — Bulk Checkpoint Import: Admin tab "Bulk Checkpoint Import" section — race selector + TSV textarea (`Code\tDisplayName\tDist`); creates each row via the existing create-checkpoint API sequentially; reports created count and per-row errors inline
- [x] 2026-06-14 — Winlink Import: excluded active checkpoint from CP selector (prevents self-import); filter: `session.Checkpoints.find(c => c.RaceID === raceID)?.CheckpointID`; tests updated to select non-active checkpoint
- [x] 2026-06-14 — Context-Sensitive Help Panel: `?` icon button in AppBar opens right-side MUI Drawer with per-tab help content; HELP array in App.tsx maps each of the 5 tab indices to a title + 3–5 items; drawer closes on backdrop click or X button; all 166 tests pass
- [x] 2026-06-14 — Tab reorder: Data Entry → Runners → Winlink Import → Winlink Export → Admin; HELP array and tab rendering updated in App.tsx
- [x] 2026-06-14 — Winlink Export: "Copy to Clipboard" renamed to "Copy Column Data"; export column header now uses CP DisplayName instead of Code; backend test assertions updated
- [x] 2026-06-14 — Pre-commit hook: `scripts/pre-commit` runs `make fmt` (fails if files changed) then `make lint`; `make install-hooks` installs it; `make install` now calls `install-hooks` so new devs get it automatically
- [x] 2026-06-14 — Responsive layout: Data Entry race cards stack full-width on mobile (xs) and side-by-side on tablet+ (sm); action cards (Log Bib, DNS/DNF, Transfer) break to column layout on xs, row on sm+; action cards changed to CSS grid `repeat(3, 1fr)` so all three remain equal-width on desktop/iPad (flex-wrap caused Transfer card to expand to full width when wrapping)
- [x] 2026-06-14 — Event Export / Import: version-tagged JSON download (admin icon button), paste-and-import in Admin; full backend service + handler + frontend API wiring + tests
- [x] 2026-06-14 — Guide tab: 6th tab with MUI Accordion sections covering Before Race Day, On Race Day, Winlink Workflow, Transferring a Runner, and Tips & Troubleshooting
- [x] 2026-06-14 — CI: re-enabled test job (removed `if: false`); added `test` to `publish` job's `needs` so container only builds when tests pass; fixed `handler_test.go` unused `mockEventExportService` type by adding handler tests for `exportEventConfig` and `importEventConfig`; added `coverage/` to ESLint ignore list
- [x] 2026-06-14 — CI: added `actions/delete-package-versions@v5` step to publish job; retains the 2 most recent sha-tagged image versions after each push; `latest` tag always protected from deletion
- [x] 2026-06-14 — Bug fix (RunnersTab HTML nesting): `<Typography variant="body2">` rendered as `<p>` wrapping a `<Chip>` (`<div>`); added `component="div"` to render as `<div>` instead, eliminating the invalid nesting and React hydration warning
- [x] 2026-06-14 — Bug fix (DataEntryTab test timing race): on slow GHA runners, the bib input appeared before the session API resolved, leaving `hasActiveCheckpoint=false`; replaced fragile `waitFor(not.toBeDisabled)` (hit 1 s ceiling) with `waitFor(() => screen.getByText('GDR'))` — the GDR race card only renders once session + races have both loaded, guaranteeing the Log button is enabled before clicking
- [x] 2026-06-14 — Bug fix (AdminTab MUI select warning): Active Event `<Select>` had `value=session.EventID` (1) before the events list finished loading, producing "out-of-range value" warnings on every test; guarded with `events.some(e => e.ID === session?.EventID)` so value stays `''` until the matching option exists; also fixed null-safety TypeScript error in export filename handler (`session.EventID` → `eventID`)
- [x] 2026-06-14 — Coverage check: all backend packages >90% (handler 97.5%, repo 97.3%, service 93.8%, mqtt 92.7%, sse 94.4%, config 93.6%); frontend 91.5% statements / 86.0% branches / 89.5% functions — all above enforced thresholds
- [x] 2026-06-14 — Bug fix (4 failing GHA tests): DataEntryTab DNS/DNF submit — `waitFor(/dns\/dnf/i)` resolved immediately (section always rendered) so Submit had `pointer-events:none`; fixed with `waitFor(() => screen.getByText('GDR'))` pattern that gates on session+races loaded. AdminTab "opens checkpoint/archive/lock-order dialog" — `screen.getByText(regex)` found both the trigger button and the dialog title; fixed with `within(screen.getByRole('dialog')).getByText(...)`.
- [x] 2026-06-14 — Bug fix (null Checkpoints crash on Winlink tabs): Go serializes nil slices as JSON `null`; when no active CPs are set `session.Checkpoints` was null in JS, crashing `.find()` in WinlinkImportTab and WinlinkExportTab. Fixed at source by initializing `Checkpoints: []entity.ActiveSessionCheckpoint{}` in the Postgres repo so it always serializes as `[]`; added defensive `?.` guards in both Winlink tab components.
- [x] 2026-06-14 — Bug fix (SSE 500 Internal Server Error): `LoggingMiddleware` wraps the `ResponseWriter` with `statusWriter` but only overrides `WriteHeader` — the SSE broker's `w.(http.Flusher)` type assertion returned `ok=false`, causing an immediate 500 on every SSE connection. Added `Flush()` (delegates to underlying writer) and `Unwrap()` (exposes underlying writer for `ResponseController`) to `statusWriter`. Also used `http.ResponseController.SetWriteDeadline(time.Time{})` in the SSE broker to disable the server's 30s write timeout for long-lived SSE connections.
- [x] 2026-06-14 — Bug fix (MUI Select out-of-range warning on Active Checkpoint): `value={activeCheckpointFor(race.ID) ?? ''}` returned a CP ID (e.g. 5) while `checkpointsByRace[race.ID]` was still loading, producing MUI "out-of-range value" warnings. Added `activeCpSelectValue(raceID)` helper that returns `''` until the matching option exists in the loaded list.
- [x] 2026-06-14 — Bug fix (MUI Tooltip disabled button warning): WinlinkExportTab Generate button was wrapped directly in `<Tooltip>` without a `<span>` intermediary; disabled buttons suppress events so MUI can't show the tooltip. Added `<span>` wrapper (all other Tooltip+disabled combos already had this).
- [x] 2026-06-14 — feat (URL-based routing): Installed `react-router-dom`; replaced `useState(tab)` with `useNavigate` + `useLocation`; each tab now has a stable URL path (`/data-entry`, `/runners`, `/winlink-import`, `/winlink-export`, `/admin`, `/guide`). `BrowserRouter` lives inside `App` so existing tests that `render(<App />)` get the router automatically — all 166 tests pass unchanged. Bare `/` redirects to `/data-entry`. No backend changes needed — `serveSPA` already falls back to `index.html` for all unmatched paths.

## Backlog

### ~~Winlink Import — Blank Line Positional Investigation~~ ✅ Resolved 2026-06-14
Root cause identified and fixed: single-digit-hour times (e.g. `7:35`) were 4 chars and failed the `len(s) >= 5` guard in `looksLikeTimeOrStatus`, causing the first data row to be skipped as a phantom header.

### ~~Frontend — Responsive Layout~~ ✅ Completed 2026-06-14
- [x] Data Entry race cards: `flex: '1 1 160px'`, `minWidth: { xs: '100%', sm: 160 }` — stack full-width on mobile, side-by-side on tablet+
- [x] Action card row (Log Bib / DNS-DNF / Transfer): changed from `md` breakpoint to `sm` so they appear side-by-side on tablet; each card has `minWidth: { xs: '100%', sm: 'auto' }` for full-width stacking on mobile

### ~~Frontend + API — Pace / Projected Arrival~~ ✅ Completed 2026-06-13

### ~~Frontend Testing~~ ✅ Completed 2026-06-14
Vitest + React Testing Library + MSW; 163 tests, 89% branch coverage (461/517), thresholds enforced in `vite.config.ts`.

### ~~Backend Testing~~ ✅ Completed 2026-06-14
All packages at >90% coverage: handler 97.3%, service 95.9%, repository 97.3%, config 93.6%, mqtt 92.7%, sse 94.4%. `make coverage` Makefile target was already wired.

### ~~UI — Bug — Data Entry cards not auto-refreshing after bib log~~ ✅ Completed 2026-06-14
- [x] Root cause: `useStream` captured handlers once at mount via empty `[]` dep array; `races` inside `onBibLogged` was always the initial empty array, so `loadLogs([])` was a no-op; fixed by storing handlers in a ref updated on every render so the EventSource callback always calls the latest closure; also added `loadLogs` calls to `submitBib` and `submitStatus` for defensive direct refresh

### ~~UI / API — Bug — Winlink import times stored as UTC, displayed as local~~ ✅ Completed 2026-06-14
- [x] Root cause: Docker container timezone is UTC; `parseTimeOfDay` built base time with `time.Local` (= UTC in Docker), storing e.g. 8:30 as 8:30 UTC → browser showed 4:30 EDT; fixed by adding `TIMEZONE` env var (IANA name, e.g. `America/New_York`), threading `*time.Location` into `WinlinkService`, and using it in `parseTimeOfDay`

### ~~UI / API — Bug — Winlink export times shifted to UTC~~ ✅ Completed 2026-06-14
- [x] Root cause: same Docker UTC issue; `RecordedAt.Local().Format("15:04")` produced UTC string (18:37) while browser showed local (14:37 EDT); fixed by using `RecordedAt.In(s.loc).Format("15:04")` with the same configured location

### ~~UI — Bulk Checkpoint Import~~ ✅ Completed 2026-06-14
- [x] Admin tab: "Bulk Checkpoint Import" section with race selector and TSV textarea (`code\tname\tdist`); creates each checkpoint via existing API sequentially; shows created count and per-row errors

### ~~UI / API — Winlink Export Email Subject Line~~ ✅ Completed 2026-06-14
- [x] Subject field appears above the column textarea after Generate; format: `<CP DisplayName> <Race Name> <HH:MM 24-hr> update`; own read-only text field + Copy Subject button; Copy Subject recomputes the time at click time for freshness

### ~~README — Developer vs. Deployment Setup Docs~~ ✅ Completed 2026-06-14
- [x] README split into Operator (Docker-only) and Developer tracks
- [x] `docker-compose.operator.yml` — pulls `ghcr.io/kbball/ares-bib-logger:latest`, operators need only Docker Desktop
- [x] `.github/workflows/ci.yml` — test + lint on PRs; build and push to GHCR on merge to main

### ~~UI — Enhancement — Rename "Copy to Clipboard" to "Copy Column Data" on Winlink Export~~ ✅ Completed 2026-06-14
- [x] Button label and tooltip updated; tests updated to match

### ~~UI — Enhancement — Reorder tabs~~ ✅ Completed 2026-06-14
- [x] New order: Data Entry → Runners → Winlink Import → Winlink Export → Admin; HELP array reordered to match; tab index mapping updated in App.tsx

### ~~UI — Enhancement — Winlink Import should exclude the active checkpoint from the CP selector~~ ✅ Completed 2026-06-14
- [x] The checkpoint dropdown on Winlink Import now excludes the station's active checkpoint for the selected race; prevents accidental self-import; filter applied via `session.Checkpoints.find(c => c.RaceID === raceID).CheckpointID`; tests updated to select Aid Station 2 (the non-active one)

### ~~UI — Enhancement — Winlink export column header should use CP display name not code~~ ✅ Completed 2026-06-14
- [x] `WinlinkService.Export` now writes `cp.DisplayName` instead of `cp.Code` as the first line; test assertions updated

### ~~UI — Context-Sensitive Help Panel~~ ✅ Completed 2026-06-14
- [x] `?` (HelpOutlined) icon button in AppBar top-right; clicking opens a right-side MUI Drawer with tab-specific help content (title + 3–5 bullet items); content updates as the active tab changes; all five tabs have written help content (Data Entry, Winlink Import, Winlink Export, Runners, Admin); Close button in drawer header; no new component file — implemented inline in App.tsx with a `HELP` array

### ~~UI — Tooltip on All Action Buttons and Icons~~ ✅ Completed 2026-06-14
- [x] Added MUI `Tooltip` to every action button and icon across all tabs

### ~~UI — Training / Onboarding Section~~ ✅ Completed 2026-06-14
- [x] "Guide" tab (6th tab) — MUI Accordion layout with 5 sections: Before Race Day, On Race Day, Winlink Workflow, Transferring a Runner Between Races, Tips & Troubleshooting; help panel entry added for Guide tab in App.tsx

### ~~UI / API — Event Export / Import~~ ✅ Completed 2026-06-14
- [x] Export: GET `/api/events/{id}/export` → version-tagged JSON (event, races, checkpoints, runners); download triggered by FileDownload icon next to active event in Admin
- [x] Import: POST `/api/events/import` → creates event + races + checkpoints + runners; Import Config section in Admin (paste JSON → Import Config button → success/error alert)
- [x] Backend: `EventExportImportService` in application layer; port interface in `domain/port/service/event_export.go`; handler in `adapter/http/handler/event_export.go`; all wired in `main.go`
- [x] Frontend: `exportEventConfig` / `importEventConfig` in `adapters/api/index.ts`

### ~~.env.example — MQTT Disabled by Default~~ ✅ Completed
- [x] `MQTT_ENABLED` default changed to `false` in `.env.example`; operators explicitly opt in when Meshtastic infrastructure is present

### ~~UI — Roster Import: Support `bib,fullName` TSV Format~~ ✅ Completed 2026-06-13
`parseTSVRoster` auto-detects 2-column (`bib\tFull Name`) vs 3-column (`bib\tfirst\tlast`); first-space split derives first/last for 2-column case.

### User Testing — MQTT Gateway and Meshtastic Messaging (Priority: Medium) 🚧 BLOCKED
- [ ] End-to-end user test of the full MQTT / Meshtastic path: Meshtastic node → gateway → Mosquitto broker → backend subscriber → bib logging
- [ ] Verify duplicate-bib detection and outbound alert publish back to the mesh
- [ ] Confirm MQTT_ENABLED=true startup, topic subscription, and graceful handling of malformed payloads
- **Blocked:** test hardware (Meshtastic nodes + gateway) not yet configured

### CI / Quality
- [x] Pre-commit hook: `scripts/pre-commit` runs `make fmt` (aborts if files changed) then `make lint`; install via `make install-hooks`; wired into `make install` so new devs get it automatically
- [x] GitHub Actions Node.js 24 migration — set `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24: true` at workflow level to opt in before the forced 2026-06-16 deadline
- [x] CI test job re-enabled; `publish` now requires both `lint` and `test` to pass before building the container
- [x] GHCR image pruning — `actions/delete-package-versions@v5` retains 2 most recent sha-tagged versions; `latest` always protected

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
| 2026-06-14 | React Router inside App (not main.tsx) | BrowserRouter placed inside App so existing tests that `render(<App />)` automatically get router context without needing a wrapper — zero test changes required |
| 2026-06-13 | Theme as `createAppTheme(mode)` factory | Single source of truth for both themes; `App.tsx` holds the `colorMode` state and passes it to `ThemeProvider` via `useMemo` |
