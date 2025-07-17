-- Decompress the hypertable
ALTER TABLE character_stats
SET (timescaledb.compress = false);
ALTER TABLE character_stats
ALTER COLUMN dps TYPE BIGINT USING dps::BIGINT;
ALTER TABLE character_stats DROP CONSTRAINT IF EXISTS character_stats_character_id_fkey;
-- -- Re-enable compression
ALTER TABLE character_stats
SET (timescaledb.compress = true);