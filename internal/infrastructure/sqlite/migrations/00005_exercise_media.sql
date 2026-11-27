-- +goose Up
-- The media keys of a library exercise from a pack, in frame order (start, peak), as a JSON list.
ALTER TABLE exercises ADD COLUMN media TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE exercises DROP COLUMN media;
