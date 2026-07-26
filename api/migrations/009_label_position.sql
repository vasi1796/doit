-- Labels gain a user-orderable position (fractional index string, same scheme
-- as lists/tasks) and an updated_at column. Position stays nullable: a
-- projection rebuild replays historical LabelCreated events that carry no
-- position, so read queries must fall back to name ordering for NULL rows.
-- +goose Up
ALTER TABLE labels ADD COLUMN position VARCHAR(255);
ALTER TABLE labels ADD COLUMN updated_at TIMESTAMPTZ;

UPDATE labels SET updated_at = created_at WHERE updated_at IS NULL;
ALTER TABLE labels ALTER COLUMN updated_at SET NOT NULL;
ALTER TABLE labels ALTER COLUMN updated_at SET DEFAULT NOW();

-- Backfill positions per user in current alphabetical order using two-char
-- printable-ASCII keys starting at '#' so fractional inserts before the first
-- item still have room ('!' < '#').
UPDATE labels l
SET position = chr(35 + (o.rn / 90)) || chr(33 + (o.rn % 90))
FROM (
    SELECT id, (row_number() OVER (PARTITION BY user_id ORDER BY name, id) - 1)::int AS rn
    FROM labels
) o
WHERE l.id = o.id;

-- +goose Down
ALTER TABLE labels DROP COLUMN updated_at;
ALTER TABLE labels DROP COLUMN position;
