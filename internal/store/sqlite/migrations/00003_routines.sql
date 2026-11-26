-- +goose Up
-- Folders group a user's routines (Push · Pull · Legs). position orders folders, and routines within a folder.
CREATE TABLE routine_folders (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    position   INTEGER NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX routine_folders_user ON routine_folders (user_id);

CREATE TABLE routines (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Deleting a folder leaves its routines in no folder.
    folder_id  TEXT REFERENCES routine_folders (id) ON DELETE SET NULL,
    name       TEXT NOT NULL,
    notes      TEXT NOT NULL DEFAULT '',
    position   INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX routines_user ON routines (user_id);

CREATE TABLE routine_exercises (
    id          TEXT PRIMARY KEY,
    routine_id  TEXT NOT NULL REFERENCES routines (id) ON DELETE CASCADE,
    -- No cascade: an exercise in use is archived, not deleted.
    exercise_id TEXT NOT NULL REFERENCES exercises (id),
    position    INTEGER NOT NULL,
    -- Group letter shared by the exercises of one superset, NULL when alone.
    superset    TEXT,
    -- Rest after each set, NULL for the app default.
    rest_s      INTEGER,
    notes       TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX routine_exercises_routine ON routine_exercises (routine_id, position);
CREATE INDEX routine_exercises_exercise ON routine_exercises (exercise_id);

-- Target sets; every target is optional.
CREATE TABLE routine_sets (
    routine_exercise_id TEXT NOT NULL REFERENCES routine_exercises (id) ON DELETE CASCADE,
    position            INTEGER NOT NULL,
    type                TEXT NOT NULL CHECK (type IN ('normal', 'warmup', 'drop', 'failure')),
    weight_g            INTEGER,
    reps_min            INTEGER,
    reps_max            INTEGER,
    duration_s          INTEGER,
    distance_m          INTEGER,
    PRIMARY KEY (routine_exercise_id, position)
) STRICT;

-- A deleted custom exercise that routines or workouts use is archived: hidden from the library, kept in history.
ALTER TABLE exercises ADD COLUMN archived_at TEXT;

-- An archived name can be reused.
DROP INDEX exercises_owner_name;
CREATE UNIQUE INDEX exercises_owner_name ON exercises (owner_id, name COLLATE NOCASE)
    WHERE owner_id IS NOT NULL AND archived_at IS NULL;

-- +goose Down
DROP INDEX exercises_owner_name;
CREATE UNIQUE INDEX exercises_owner_name ON exercises (owner_id, name COLLATE NOCASE) WHERE owner_id IS NOT NULL;
ALTER TABLE exercises DROP COLUMN archived_at;
DROP TABLE routine_sets;
DROP TABLE routine_exercises;
DROP TABLE routines;
DROP TABLE routine_folders;
