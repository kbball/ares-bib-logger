-- Enum: runner lifecycle status
CREATE TYPE runner_status AS ENUM (
    'UNKNOWN',
    'ACTIVE',
    'DNS',
    'DNF',
    'FINISHED',
    'MOVED'
);

-- Enum: source of a checkpoint log entry
CREATE TYPE log_source AS ENUM (
    'MESHTASTIC',
    'MANUAL',
    'WINLINK_IMPORT'
);

-- Events (e.g. "GA Death Race", "GA Jewel")
CREATE TABLE events (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT events_name_unique UNIQUE (name)
);

-- Races within an event (e.g. "100 Miler", "GDR")
CREATE TABLE races (
    id             SERIAL PRIMARY KEY,
    event_id       INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    -- locked after the roster has been imported; enforced at the API layer
    roster_locked  BOOLEAN NOT NULL DEFAULT FALSE,
    -- locked once the race starts; prevents checkpoint order changes that would
    -- break positional Winlink import mappings
    order_locked   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT races_event_name_unique UNIQUE (event_id, name)
);

-- Checkpoints per race, in display order
-- For 100 Miler: code encodes direction, e.g. "StoverRoadOut BoundDepart"
CREATE TABLE checkpoints (
    id            SERIAL PRIMARY KEY,
    race_id       INTEGER NOT NULL REFERENCES races(id) ON DELETE CASCADE,
    code          TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    display_order SMALLINT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT checkpoints_race_code_unique  UNIQUE (race_id, code),
    CONSTRAINT checkpoints_race_order_unique UNIQUE (race_id, display_order)
);

-- Runners: pre-loaded from roster before the race
-- sort_order preserves the original spreadsheet row order (drives Winlink export)
-- When a runner transfers races: status = MOVED in original race;
--   a new row is inserted at the bottom of the target race
CREATE TABLE runners (
    id         SERIAL PRIMARY KEY,
    race_id    INTEGER NOT NULL REFERENCES races(id) ON DELETE CASCADE,
    bib_number INTEGER NOT NULL,
    name       TEXT NOT NULL,
    sort_order INTEGER NOT NULL,
    status     runner_status NOT NULL DEFAULT 'UNKNOWN',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT runners_race_bib_unique   UNIQUE (race_id, bib_number),
    CONSTRAINT runners_race_order_unique UNIQUE (race_id, sort_order)
);

-- One log entry per runner per checkpoint
-- Duplicate (runner_id, checkpoint_id) is prevented at the DB level
CREATE TABLE checkpoint_logs (
    id            SERIAL PRIMARY KEY,
    runner_id     INTEGER NOT NULL REFERENCES runners(id) ON DELETE CASCADE,
    checkpoint_id INTEGER NOT NULL REFERENCES checkpoints(id) ON DELETE CASCADE,
    recorded_at   TIMESTAMPTZ NOT NULL,
    source        log_source NOT NULL,
    raw_message   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT checkpoint_logs_runner_checkpoint_unique UNIQUE (runner_id, checkpoint_id)
);

-- Singleton active session (id is always 1)
-- Stores which event and checkpoints are currently active at this station
CREATE TABLE active_session (
    id         INTEGER PRIMARY KEY DEFAULT 1,
    event_id   INTEGER REFERENCES events(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT active_session_singleton CHECK (id = 1)
);

INSERT INTO active_session (id) VALUES (1);

-- One active checkpoint per race in the current session
-- For the 100 Miler, this row is updated when direction switches (Out→In)
CREATE TABLE active_session_checkpoints (
    session_id    INTEGER NOT NULL REFERENCES active_session(id) ON DELETE CASCADE,
    race_id       INTEGER NOT NULL REFERENCES races(id) ON DELETE CASCADE,
    checkpoint_id INTEGER NOT NULL REFERENCES checkpoints(id) ON DELETE CASCADE,
    PRIMARY KEY (session_id, race_id)
);
