-- +goose Up
-- Library exercises (owner_id NULL, slug from the dataset) and users' custom ones.
CREATE TABLE exercises (
    id                TEXT PRIMARY KEY,
    owner_id          TEXT REFERENCES users (id) ON DELETE CASCADE,
    slug              TEXT UNIQUE,
    name              TEXT NOT NULL,
    kind              TEXT NOT NULL CHECK (kind IN ('weight_reps', 'reps_only', 'weighted_bodyweight',
                                                    'assisted_bodyweight', 'duration', 'distance_duration')),
    category          TEXT NOT NULL,
    equipment         TEXT NOT NULL,
    primary_muscles   TEXT NOT NULL DEFAULT '[]',
    secondary_muscles TEXT NOT NULL DEFAULT '[]',
    force             TEXT NOT NULL DEFAULT '',
    level             TEXT NOT NULL DEFAULT '',
    mechanic          TEXT NOT NULL DEFAULT '',
    instructions      TEXT NOT NULL DEFAULT '[]',
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    CHECK ((owner_id IS NULL) = (slug IS NOT NULL))
) STRICT;

-- shortcut: NOCASE folds ASCII only ("Élan" and "élan" can coexist); use a normalised name column if that bites.
CREATE UNIQUE INDEX exercises_owner_name ON exercises (owner_id, name COLLATE NOCASE) WHERE owner_id IS NOT NULL;

-- Small key/value state, e.g. the version of the exercise library last synced.
CREATE TABLE meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;

-- +goose Down
DROP TABLE meta;
DROP TABLE exercises;
