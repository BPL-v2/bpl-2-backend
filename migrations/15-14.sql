ALTER TABLE objectives
    RENAME COLUMN parent_id to category_id;
ALTER TABLE objectives DROP COLUMN event_id;