ALTER TABLE scoring_presets
ADD COLUMN extra text NULL;
ALTER TABLE scoring_presets
DROP COLUMN IF EXISTS "type";