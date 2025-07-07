-- Decompress the hypertable
ALTER TABLE character_stats
SET (timescaledb.compress = false);
ALTER TABLE character_stats DROP CONSTRAINT IF EXISTS character_stats_character_id_fkey;
ALTER TABLE characters
ALTER COLUMN "id" TYPE TEXT;
ALTER TABLE character_stats
ALTER COLUMN character_id TYPE TEXT;
ALTER TABLE character_stats
ADD CONSTRAINT character_stats_character_id_fkey FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE;
CREATE TABLE IF NOT EXISTS character_pobs (
  id bigserial not NULL,
  character_id TEXT NOT NULL,
  export TEXT NOT NULL,
  "level" INT NOT NULL,
  ascendancy TEXT NOT NULL,
  main_skill TEXT NOT NULL,
  "timestamp" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
);
-- -- Re-enable compression
ALTER TABLE character_stats
SET (timescaledb.compress = true);