-- Decompress the hypertable
ALTER TABLE character_stats
SET (timescaledb.compress = false);
ALTER TABLE character_stats
ADD COLUMN movement_speed INTEGER NOT NULL DEFAULT 0;
-- -- Re-enable compression
ALTER TABLE character_stats
SET (timescaledb.compress = true);