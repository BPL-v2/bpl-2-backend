-- Description: Add event_id to scoring_categories and add public and locked columns to events
ALTER TABLE scoring_categories
ADD event_id int8 NULL;
-- Fill event_id with the event_id of the parent category
WITH RECURSIVE connected_categories AS (
    SELECT sc.id,
        sc.parent_id,
        e.id AS event_id
    FROM scoring_categories sc
        JOIN events e ON sc.id = e.scoring_category_id
    UNION ALL
    SELECT sc.id,
        sc.parent_id,
        cc.event_id
    FROM scoring_categories sc
        INNER JOIN connected_categories cc ON sc.parent_id = cc.id
)
UPDATE scoring_categories
SET event_id = cc.event_id
FROM connected_categories cc
WHERE scoring_categories.id = cc.id;
-- Remove orphaned categories
DELETE FROM scoring_categories
WHERE event_id IS NULL;
-- Add not null constraint
ALTER TABLE scoring_categories
ALTER COLUMN event_id
SET NOT NULL;
-- Add foreign key constraint
ALTER TABLE scoring_categories
ADD CONSTRAINT fk_scoring_categories_event_id FOREIGN KEY (event_id) REFERENCES events (id);
-- Drop scoring_category_id column
ALTER TABLE events DROP COLUMN scoring_category_id;
-- Add new columns to events
ALTER TABLE events
ADD COLUMN public BOOLEAN NULL;
--
ALTER TABLE events
ADD COLUMN locked BOOLEAN NULL;
-- Add default values
UPDATE events
SET public = TRUE,
    locked = FALSE;
--
ALTER TABLE events
ALTER COLUMN public
SET NOT NULL;
--
ALTER TABLE events
ALTER COLUMN locked
SET NOT NULL;