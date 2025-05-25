ALTER TABLE objectives
    RENAME COLUMN category_id to parent_id;
ALTER TABLE objectives
ADD event_id int8;
ALTER TABLE objectives
ALTER COLUMN parent_id DROP NOT NULL;
ALTER TABLE objectives DROP CONSTRAINT IF EXISTS fk_bpl2_scoring_categories_objectives;