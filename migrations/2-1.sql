-- add back scoring_category_id to events
ALTER TABLE events
ADD COLUMN scoring_category_id int8 NULL;
-- get root category ids for each event
UPDATE events e
SET scoring_category_id = sc.id
FROM scoring_categories sc
WHERE sc.event_id = e.id
    AND sc.parent_id IS NULL;
-- Add foreign key constraint
ALTER TABLE events
ADD CONSTRAINT fk_events_scoring_category_id FOREIGN KEY (scoring_category_id) REFERENCES scoring_categories (id);
-- Drop event_id column
ALTER TABLE scoring_categories DROP COLUMN event_id;
-- Drop public and locked columns
ALTER TABLE events DROP COLUMN public;
ALTER TABLE events DROP COLUMN locked;