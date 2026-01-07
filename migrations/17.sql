ALTER TABLE scoring_presets
ALTER COLUMN extra TYPE jsonb USING extra::jsonb;
UPDATE scoring_presets SET extra = '{}' WHERE extra IS NULL;