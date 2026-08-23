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
- **`query <bib>` command** (both MQTT/Meshtastic and MeshCore): checked before bib parsing on every inbound text message, so it never falls through and gets logged as a bib itself. Replies over the same mesh with a compact one-line summary — status, name, last known checkpoint + time, and pace (when computable) — e.g. `"101 Jane Doe: ACTIVE last AS4 14:32 pace 12:30/mi"`, or `"101 not found"` for an unrecognized bib. Reuses each adapter's existing ack/reply publish mechanism (`publishText`). Last-known-checkpoint is always reported regardless of runner status (including DNS/DNF) since locating a dropped runner is the primary emergency-comms use case; pace is omitted for terminal statuses and when fewer than 2 distance-tagged checkpoints have been logged.

**Fallback / manual-entry mode:**
- Controlled by `MQTT_ENABLED` env var (default `true`)
- When `false`: MQTT adapter does not start; app runs fully on manual entry via UI
- No degradation to other features — all UI tabs, Winlink import/export, and tabular view work normally
- Useful when Meshtastic infrastructure is unavailable or being tested without a gateway

### 2. Admin Panel (UI)
Three sections, grouped into two collapsed-by-default accordions: **Setup** (Active Event, Roster Import, Bulk Checkpoint Import, Races) and **Edit Runners** (Change Runner Status) — keeps the page short on load; operators expand only what they need.

**Event & checkpoint configuration**
- Select active event (GDR or GA Jewel)
- For each race: set the active checkpoint ID for this station (dropdown of checkpoints for that race)
- For 100M: checkpoint dropdown handles the Out/In switch — operator just picks the new checkpoint mid-race
- Settings persist in ActiveSession (survive restarts) — safe to update mid-race
- Per-event toggle: "blank line between header and first row (Winlink)" — some stations' Winlink convention has a blank row between the header and first data row, some don't; `Event.WinlinkBlankLineAfterHeader` controls this for both Export and Import/Preview for that event. Import only skips the blank line when a header was actually detected *and* the next line really is blank, so a mismatched toggle never eats a real data row.

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
- **(Planned, not yet built)** Overall stats card: same fields as the per-race cards, summed across all races in the active event (GA Jewel's 4 races combined; GDR shows as a single card already so this is a no-op there). Projected next arrival is per-checkpoint/per-race, so the overall card omits it or notes it's not meaningful in aggregate — a per-implementation detail to resolve, not a plan-level decision. See Backlog.
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
- Pre-write preview/confirm step: submit first calls `POST /api/winlink/import/preview` (dry-run, no writes) via `WinlinkService.Preview`; a 100% clean parse (zero skips) imports immediately as before, otherwise a confirm modal shows the full per-row breakdown (position, bib, Create/Update/Skip, value or skip reason) before the operator commits or cancels

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
- Status filter: multi-select chips (ACTIVE/DNS/DNF/FINISHED/MOVED/UNKNOWN); combines with search and applies on both the All tab and individual race tabs
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

**v1.0 — 2026-06-13 through 2026-06-14**

- **Foundation**: Stack, architecture, and scaffolding — Go/TypeScript/React/Postgres/Docker; CLAUDE.md, Makefile, docker-compose (with Mosquitto), go.mod, frontend init (Vite/MUI/Vitest/ESLint/Prettier)
- **Analysis**: GDR spreadsheet (Winlink format, column layout, roster structure) and GA Jewel spreadsheet (4 races, bib ranges, checkpoint chains, Out/In 100M direction); domain model and feature spec captured in plan
- **Backend**: All domain entities, port interfaces, application services, Postgres repos (golang-migrate), HTTP handlers, MQTT adapter, Winlink import/export, roster importer; archive event, lock checkpoint order, DNS/DNF Winlink import, pace/projected arrival (migration 000003 + full stack), Change Runner Status, Event export/import, Winlink import upsert
- **Frontend**: Material UI, six tabs (Data Entry, Runners, Winlink Import, Winlink Export, Admin, Guide), SSE real-time updates, light/dark toggle, runner detail modal, responsive layout, context-sensitive help panel, tooltips on all actions, URL-based tab routing (React Router), Winlink export email subject, bulk checkpoint import, logo
- **Testing**: Frontend 163 tests / 89% branch coverage (Vitest + RTL + MSW); backend all packages >90% (handler 97.5%, repo 97.3%, service 93.8%, mqtt 92.7%, sse 94.4%, config 93.6%); thresholds enforced in both stacks
- **CI/CD**: GitHub Actions lint+test on PR, build+push to GHCR on merge; docker-compose.operator.yml for operators; pre-commit hooks (fmt + lint); README split into Operator/Developer tracks
- **Bug fixes**: Winlink blank-line positional shift (single-digit-hour time parsing); Docker timezone (TIMEZONE env var, WinlinkService); SSE flusher/write-deadline; Data Entry SSE closure stale state; null Checkpoints crash on Winlink tabs; MUI out-of-range Select warnings (session load race); HTML nesting (Typography/Chip); GHA test timing races

**v1.1 — 2026-08-23**

- Runners tab: multi-select status filter (chips), combining with search and race tabs
- Winlink import preview/confirm step: `WinlinkService.Preview` (read-only, shares row classification with `Import` via new `parseImportRows`), `POST /api/winlink/import/preview`, frontend auto-imports on a clean parse and otherwise shows a confirm modal with the full per-row breakdown before committing
- Mesh `query <bib>` command: new `backend/internal/domain/pace` package (Go port of `frontend/src/domain/pace.ts`); `CheckpointLogService.QueryRunner` assembles a compact status/last-station/pace reply; both MQTT/Meshtastic and MeshCore adapters detect the command ahead of bib parsing and reply over the mesh via a shared `publishText` helper
- Winlink blank-line-after-header, event-configurable: migration 000005 adds `Event.WinlinkBlankLineAfterHeader`; `WinlinkService` resolves it per-race via race→event lookup and both `Export` and `parseImportRows` (shared by `Import`/`Preview`) respect it; new `PUT /api/events/{id}/winlink-format` endpoint; Admin panel toggle on the active event
- Admin panel restructured into two collapsed-by-default accordions ("Setup", "Edit Runners")
- Dark/light mode now persists across refreshes (`localStorage`, fails open if storage is unavailable); added a `window.localStorage` stub to the frontend test setup since this project's jsdom test environment doesn't implement it
- "Column Name" field on checkpoints: migration 000006 adds `checkpoints.column_name` (nullable); `Checkpoint.ColumnName *string` threaded through repo/service/handler `Create`/`Update` and the event export/import DTO; `WinlinkService.Export` uses it for the header line, falling back to `DisplayName` when unset or blank; Admin panel checkpoint create/edit forms gained a "Column Name" input (table column + inline edit)
- Winlink import header validation: `WinlinkService.Preview` now also fetches the target checkpoint and compares the pasted text's header line (when present) against the checkpoint's expected header (`ColumnName`, falling back to `DisplayName`) via new `checkpointHeader`/`pastedHeaderLine` helpers; `WinlinkPreviewResult` gained `HeaderMismatch`/`PastedHeader`/`ExpectedHeader`; frontend now shows the confirm modal (with a warning `Alert`) whenever there's a header mismatch, even on an otherwise-clean parse that would normally auto-import
- Correct a mis-logged bib (Admin → Edit Runners): migration 000007 adds `CORRECTION` to the `log_source` enum; `CheckpointLogRepository.Delete(runnerID, checkpointID)` (new); `CheckpointLogService.CorrectLog` (parses `HH:MM`/`HH:MM:SS` via shared `parseWallClockTime`, upserts with `Source: CORRECTION`, wakes `UNKNOWN` runners to `ACTIVE`) and `.DeleteLog` (looks up the runner by bib within the race, deletes the log); new `POST /api/log/correction` and `DELETE /api/log/correction` endpoints (JSON body: `race_id`, `checkpoint_id`, `bib_number`, and `time` for the POST); `CheckpointLogService` now takes a `*time.Location` constructor param (same timezone source as `WinlinkService`); Admin panel "Edit Runners" accordion gained "Manually Log a Bib" and "Remove a Checkpoint Log" (with delete confirmation dialog) sections; `del()` API client helper gained an optional JSON body for the DELETE-with-body call

## Backlog

Ordered by priority (2026-08-23):

### ~~0. Bulk checkpoint import missing Column Name field~~ ✅ DONE
- Bulk Import of checkpoints now supports an optional 4th TSV column, `ColumnName` (`Code, DisplayName, DistFromStart, ColumnName`), passed through to the existing `createCheckpoint` API (backend already supported it end-to-end)

### 0. Admin page doesn't unload event data on archive
- When archiving an event, the admin page should refresh/unload the data associated with that event — currently races stay loaded until a new event is created and selected
- Not yet implemented — captured here for future work

### 0. Manually Log a Bib — date association for timestamp
- When logging using the timestamp on "Manually Log a Bib" (admin page), determine whether a date is/should be associated with the entered time, and add it to the form if needed
- Not yet implemented — captured here for future work

### 0. Optional cutoff time on checkpoint configuration
- Add a cutoff time column to aid station / checkpoint configuration — optional, since not all aid stations have a cutoff
- Not yet implemented — captured here for future work

### 0. Persist last-opened admin accordion across navigation
- On the Admin page, persist which accordion was last opened so it stays open if the user clicks off the page and returns
- Not yet implemented — captured here for future work

### 1. Add single runner to roster (late race addition)
- Admin action: add one runner directly to a race's roster, appended to the bottom (sort_order = max existing + 1)
- Reasoning: covers a runner who registers late and isn't in the pre-loaded roster or any other race — distinct from [Runner Race Transfer](#7-runner-race-transfer), which moves an existing runner between races
- Not yet implemented — captured here for future work

### 2. Overall stats card on Data Entry tab
- Add a card alongside the existing per-race stat cards showing totals across all races in the active event: total starters, on-course, DNS, DNF, finishers
- Same underlying data as the per-race cards, just summed; primarily useful for GA Jewel (4 concurrent races) — GDR already effectively shows one card
- Not yet implemented — captured here for future work

### 3. Split pace between aid stations on runner detail modal
- The runner detail modal (`RunnersTab.tsx`, Checkpoint Log table) currently shows Checkpoint + Time columns only
- Add a "Split pace" column: pace between each pair of consecutive logged checkpoints that both have a known `DistanceFromStart`, mirroring the distance/time-delta math already in `computeRunnerPace`/`formatPace` (`frontend/src/domain/pace.ts`) but computed for every consecutive pair in the table (a full splits view), not just the last two checkpoints used for the live "Pace" stat elsewhere
- Blank/— for the first logged checkpoint (no prior split) and for any pair where either checkpoint lacks a distance
- Not yet implemented — captured here for future work

### User Testing — MQTT Gateway and Mesh Messaging ✅ COMPLETE
- [x] End-to-end user test of the full MQTT / Meshtastic path: Meshtastic node → gateway → Mosquitto broker → backend subscriber → bib logging
- [x] End-to-end user test of the full MeshCore path: MeshCore node → meshcore-mqtt bridge → Mosquitto broker → backend subscriber → bib logging
- [x] Verify duplicate-bib detection and outbound ack back to the mesh (both technologies)
- [x] Confirm MQTT_ENABLED=true startup, topic subscription, and graceful handling of malformed payloads

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
| 2026-08-23 | Runners tab status filter is multi-select, client-side | Runner data is already fully loaded client-side for the tab; multi-select chips let an operator combine statuses (e.g. DNS + DNF) without new API params |
| 2026-08-23 | `LastLoggedCheckpoint` reports a runner's last station regardless of status | `ComputeRunnerPace` intentionally returns empty for DNS/DNF/MOVED/FINISHED (pace isn't meaningful once stopped) — but a dropped runner's last known location is exactly what a search-and-rescue mesh query needs, so it must not be gated by the same status check |
| 2026-08-23 | `query <bib>` checked before bib-list parsing, not after | Both mesh adapters previously treated every non-numeric token as noise to skip; without an explicit early check, `"query 101"` would silently log bib 101 as a real checkpoint hit instead of being recognized as a command |
| 2026-08-23 | Winlink blank-line-after-header is a per-event setting, resolved via race→event | Not tied to any one race/checkpoint — it's a formatting convention for the whole event's Winlink traffic; `Import`/`Preview` look it up from `raceID` rather than `ActiveSession` so they stay self-contained (no new session dependency) |
| 2026-08-23 | Blank line only consumed when actually present, even if the event flag is on | A mismatched setting (flag enabled, but this particular paste has no blank line) must never eat a real data row — `parseImportRows` checks the line content, not just the flag, before advancing past it |
| 2026-08-23 | Color mode persistence fails open (try/catch + fallback to light) instead of assuming `localStorage` exists | Private browsing and disabled storage can make `localStorage` unavailable or throw; the app must still boot and toggle themes in that case, just without persistence |
| 2026-08-23 | `Checkpoint.ColumnName` is a nullable `*string`, falls back to `DisplayName` in `WinlinkService.Export` | Most stations' internal display name already matches their spreadsheet header — only override it when the two diverge; blank/unset never breaks export |
| 2026-08-23 | Winlink import header validation only warns (never blocks) and is skipped entirely when no header line is present | A wrong-checkpoint paste is an operator error worth flagging, but the operator may have a legitimate reason to proceed (e.g. a renamed checkpoint); the existing preview/confirm modal is the natural place to surface it rather than a hard stop |
| 2026-08-23 | Manual bib correction uses a new `CORRECTION` log source rather than reusing `MANUAL` | Keeps a post-hoc fix distinguishable in the audit trail from a genuine at-the-time manual entry, without overloading `RawMessage` to carry that distinction |
| 2026-08-23 | `CorrectLog`/`DeleteLog` find the runner by listing the race's roster and matching bib number, instead of adding a new `RunnerRepository.GetByBibInRace` method | The operator already picks an explicit race in the UI (unlike mesh logging, which only has an event-wide active session), and races are small enough that an in-memory scan mirrors the pattern `WinlinkService` already uses for its `byOrder` map — no new repo method needed |
| 2026-08-23 | `parseTimeOfDay` promoted to a package-level `parseWallClockTime(loc, str)` function shared by `WinlinkService` and `CheckpointLogService` | Both need identical wall-clock-on-today's-date parsing in the configured timezone; a shared free function avoids duplicating the logic while keeping each service's public API unchanged |
