ALTER TABLE character_pobs
RENAME COLUMN timestamp TO created_at;
ALTER TABLE character_pobs
ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT NOW();

UPDATE character_pobs
SET updated_at = created_at;