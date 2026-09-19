ALTER TABLE races ADD COLUMN roster_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE races ADD COLUMN winlink_footer_rows INTEGER NOT NULL DEFAULT 0;

-- Grandfather in any runners a locked race already has as part of its
-- "original" list, so this migration never turns pre-existing rows into
-- footer entries. Only additions made after this migration land in the footer.
UPDATE races
SET roster_count = COALESCE((SELECT COUNT(*) FROM runners WHERE runners.race_id = races.id), 0)
WHERE roster_locked = true;
