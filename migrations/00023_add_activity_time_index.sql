-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_activity_time ON activity (time);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_activity_time;
