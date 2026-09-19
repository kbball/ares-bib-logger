ALTER TABLE events ADD COLUMN winlink_reminder_minutes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE races ADD COLUMN winlink_last_export_at TIMESTAMPTZ;
ALTER TABLE races ADD COLUMN winlink_reminder_dismissed BOOLEAN NOT NULL DEFAULT FALSE;
