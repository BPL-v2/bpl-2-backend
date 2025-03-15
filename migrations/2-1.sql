-- add back scoring_category_id to events
ALTER TABLE bpl2.events
ADD COLUMN scoring_category_id int8 NULL;
-- get root category ids for each event
UPDATE bpl2.events e
SET scoring_category_id = sc.id
FROM bpl2.scoring_categories sc
WHERE sc.event_id = e.id
    AND sc.parent_id IS NULL;
-- Add not null constraint
ALTER TABLE bpl2.events
ALTER COLUMN scoring_category_id
SET NOT NULL;
-- Add foreign key constraint
ALTER TABLE bpl2.events
ADD CONSTRAINT fk_events_scoring_category_id FOREIGN KEY (scoring_category_id) REFERENCES bpl2.scoring_categories (id);
-- Drop event_id column
ALTER TABLE bpl2.scoring_categories DROP COLUMN event_id;
-- Drop public and locked columns
ALTER TABLE bpl2.events DROP COLUMN public;
ALTER TABLE bpl2.events DROP COLUMN locked;