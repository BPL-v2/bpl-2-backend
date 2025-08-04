ALTER TABLE change_ids
ADD COLUMN IF NOT EXISTS "timestamp" timestamptz NULL;