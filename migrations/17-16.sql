-- Drop unique constraints
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_user_objective_timestamp_unique;
-- Drop foreign keys
ALTER TABLE objectives DROP CONSTRAINT IF EXISTS objectives_event_id_fkey;
ALTER TABLE objectives DROP CONSTRAINT IF EXISTS objectives_parent_id_fkey;
ALTER TABLE objective_matches DROP CONSTRAINT IF EXISTS objective_matches_stash_change_id_fkey;
ALTER TABLE objective_matches DROP CONSTRAINT IF EXISTS objective_matches_event_id_fkey;
ALTER TABLE ladder_entries DROP CONSTRAINT IF EXISTS ladder_entries_user_id_fkey;
ALTER TABLE ladder_entries DROP CONSTRAINT IF EXISTS ladder_entries_event_id_fkey;
ALTER TABLE scoring_presets DROP CONSTRAINT IF EXISTS scoring_presets_event_id_fkey;
ALTER TABLE signups DROP CONSTRAINT IF EXISTS signups_event_id_fkey;
ALTER TABLE signups DROP CONSTRAINT IF EXISTS signups_user_id_fkey;
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_event_id_fkey;
ALTER TABLE teams DROP CONSTRAINT IF EXISTS team_event_id_fkey;
ALTER TABLE teams DROP CONSTRAINT IF EXISTS team_users_team_id_fkey;
ALTER TABLE teams DROP CONSTRAINT IF EXISTS team_users_user_id_fkey;
-- Drop indexes
DROP INDEX IF EXISTS objectives_event_id_idx;
DROP INDEX IF EXISTS stash_changes_stash_id_idx;
DROP INDEX IF EXISTS scoring_presets_event_id_idx;
DROP INDEX IF EXISTS signups_event_id_idx;
DROP INDEX IF EXISTS signups_user_id_idx;
DROP INDEX IF EXISTS submissions_event_id_idx;
DROP INDEX IF EXISTS team_suggestions_team_id_idx;
DROP INDEX IF EXISTS team_event_id_idx;
-- Drop primary keys
ALTER TABLE team_suggestions DROP CONSTRAINT IF EXISTS team_suggestions_pkey;
CREATE INDEX stash_changes_id_idx ON stash_changes (id);
ALTER TABLE stash_changes DROP CONSTRAINT IF EXISTS stash_changes_pkey;
-- make user_id not nullable in ladder_entries
UPDATE ladder_entries
SET user_id = 0
WHERE user_id is NULL;
ALTER TABLE ladder_entries
ALTER COLUMN user_id
SET NOT NULL;