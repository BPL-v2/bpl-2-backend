-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_character_pobs_character_created ON character_pobs (character_id, created_at DESC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_character_pobs_character_created;
