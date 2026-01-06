-- Migration 15: Change scoring_id to many-to-many relationship with junction table

-- Create junction table for objectives and scoring presets
CREATE TABLE objective_scoring_presets (
    objective_id int4 NOT NULL,
    scoring_preset_id int4 NOT NULL,
    PRIMARY KEY (objective_id, scoring_preset_id),
    CONSTRAINT fk_objective_scoring_presets_objective FOREIGN KEY (objective_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_objective_scoring_presets_scoring_preset FOREIGN KEY (scoring_preset_id) REFERENCES scoring_presets(id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create index for reverse lookups
CREATE INDEX objective_scoring_presets_scoring_preset_id_idx 
    ON objective_scoring_presets USING btree (scoring_preset_id);

-- Migrate existing data: insert non-NULL scoring_id values into junction table
INSERT INTO objective_scoring_presets (objective_id, scoring_preset_id)
SELECT id, scoring_id 
FROM objectives 
WHERE scoring_id IS NOT NULL;

-- Drop the foreign key constraint and old column
ALTER TABLE objectives 
DROP CONSTRAINT IF EXISTS fk_bpl2_objectives_scoring_preset;

ALTER TABLE objectives 
DROP COLUMN scoring_id;
