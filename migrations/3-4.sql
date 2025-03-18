ALTER TABLE teams
ADD COLUMN color text;
UPDATE teams
SET color = '';
ALTER TABLE teams
ALTER COLUMN color
SET NOT NULL;