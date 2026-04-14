-- +goose Up
ALTER TABLE character_pobs ADD COLUMN high_ilevel_flasks int2 NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE character_pobs DROP COLUMN high_ilevel_flasks;
