-- Re-pin the 009 label position backfill to byte ordering (COLLATE "C") so it
-- matches the client-side Dexie backfill (JS code-unit comparison) regardless
-- of the database locale — under glibc locales, ORDER BY name diverges from
-- the client and the two sides would never converge (no events exist for
-- backfills). Applied in the same release as 009, so no user reorders can
-- exist between the two. Keys stay within printable ASCII up to 8,280 labels
-- per user — far beyond realistic use.
-- +goose Up
UPDATE labels l
SET position = chr(35 + (o.rn / 90)) || chr(33 + (o.rn % 90))
FROM (
    SELECT id, (row_number() OVER (PARTITION BY user_id ORDER BY name COLLATE "C", id) - 1)::int AS rn
    FROM labels
) o
WHERE l.id = o.id;

-- +goose Down
UPDATE labels l
SET position = chr(35 + (o.rn / 90)) || chr(33 + (o.rn % 90))
FROM (
    SELECT id, (row_number() OVER (PARTITION BY user_id ORDER BY name, id) - 1)::int AS rn
    FROM labels
) o
WHERE l.id = o.id;
