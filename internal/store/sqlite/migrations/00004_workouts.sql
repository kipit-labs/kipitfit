-- +goose Up
-- Finished workouts. The id is made on the phone when the workout starts, so a finish sent twice saves once.
CREATE TABLE workouts (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- The routine it was started from; a deleted routine leaves its workouts.
    routine_id TEXT REFERENCES routines (id) ON DELETE SET NULL,
    title      TEXT NOT NULL,
    notes      TEXT NOT NULL DEFAULT '',
    started_at TEXT NOT NULL,
    ended_at   TEXT NOT NULL,
    -- Σ weight × reps of the sets that aren't warm-ups, in grams.
    volume_g   INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX workouts_user_started ON workouts (user_id, started_at);
CREATE INDEX workouts_routine ON workouts (routine_id);

CREATE TABLE workout_exercises (
    id          TEXT PRIMARY KEY,
    workout_id  TEXT NOT NULL REFERENCES workouts (id) ON DELETE CASCADE,
    -- No cascade: an exercise in use is archived, not deleted.
    exercise_id TEXT NOT NULL REFERENCES exercises (id),
    position    INTEGER NOT NULL,
    superset    TEXT,
    notes       TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE INDEX workout_exercises_workout ON workout_exercises (workout_id, position);
CREATE INDEX workout_exercises_exercise ON workout_exercises (exercise_id);

CREATE TABLE sets (
    workout_exercise_id TEXT NOT NULL REFERENCES workout_exercises (id) ON DELETE CASCADE,
    position            INTEGER NOT NULL,
    type                TEXT NOT NULL CHECK (type IN ('normal', 'warmup', 'drop', 'failure')),
    weight_g            INTEGER,
    reps                INTEGER,
    -- RPE × 10 (8.5 is 85), so no floats.
    rpe                 INTEGER CHECK (rpe BETWEEN 60 AND 100),
    duration_s          INTEGER,
    distance_m          INTEGER,
    PRIMARY KEY (workout_exercise_id, position)
) STRICT;

-- +goose Down
DROP TABLE sets;
DROP TABLE workout_exercises;
DROP TABLE workouts;
