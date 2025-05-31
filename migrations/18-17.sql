DROP TABLE change_ids;
ALTER TABLE stash_changes CREATE COLUMN next_change_id text NULL;