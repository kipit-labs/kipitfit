-- +goose Up
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    is_admin      INTEGER NOT NULL DEFAULT 0,
    -- '' until picked: the app follows the browser.
    language      TEXT NOT NULL DEFAULT '' CHECK (language IN ('', 'en', 'fr')),
    unit          TEXT NOT NULL DEFAULT 'kg' CHECK (unit IN ('kg', 'lb')),
    created_at    TEXT NOT NULL
) STRICT;

-- Only a SHA-256 of the session token is stored.
CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
) STRICT;

CREATE INDEX sessions_user_id ON sessions (user_id);

-- Invite links, copied by hand to the person; only a SHA-256 of the token is stored.
CREATE TABLE invites (
    id         TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    email      TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    used_at    TEXT
) STRICT;

-- +goose Down
DROP TABLE invites;
DROP TABLE sessions;
DROP TABLE users;
