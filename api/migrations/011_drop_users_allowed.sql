-- users.allowed was written as true for every account and never read, so it
-- looked like an authorisation gate while enforcing nothing. Access control is
-- the ALLOWED_EMAILS allowlist, checked at login.
-- +goose Up
ALTER TABLE users DROP COLUMN allowed;

-- +goose Down
ALTER TABLE users ADD COLUMN allowed BOOLEAN NOT NULL DEFAULT false;
UPDATE users SET allowed = true;
