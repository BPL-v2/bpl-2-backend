-- missing primary keys
ALTER TABLE team_suggestions
ADD PRIMARY KEY (id, team_id);
DROP INDEX stash_changes_id_idx;
ALTER TABLE stash_changes
ADD PRIMARY KEY (id);
-- missing indexes
CREATE INDEX objectives_event_id_idx ON objectives (event_id);
CREATE INDEX stash_changes_stash_id_idx ON stash_changes (stash_id);
CREATE INDEX scoring_presets_event_id_idx ON scoring_presets (event_id);
CREATE INDEX signups_event_id_idx ON signups (event_id);
CREATE INDEX signups_user_id_idx ON signups (user_id);
CREATE INDEX submissions_event_id_idx ON submissions (event_id);
CREATE INDEX team_suggestions_team_id_idx ON team_suggestions (team_id);
CREATE INDEX team_event_id_idx ON teams (event_id);
-- missing foreign keys
ALTER TABLE objectives
ADD CONSTRAINT objectives_event_id_fkey FOREIGN KEY (event_id) REFERENCES events (id);
ALTER TABLE objectives
ADD CONSTRAINT objectives_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES objectives (id);
ALTER TABLE objective_matches
ADD CONSTRAINT objective_matches_stash_change_id_fkey FOREIGN KEY (stash_change_id) REFERENCES stash_changes (id);
ALTER TABLE objective_matches
ADD CONSTRAINT objective_matches_event_id_fkey FOREIGN KEY (event_id) REFERENCES events (id);
ALTER TABLE ladder_entries
ALTER COLUMN user_id DROP NOT NULL;
UPDATE ladder_entries
SET user_id = NULL
WHERE user_id = 0;
ALTER TABLE ladder_entries
ADD CONSTRAINT ladder_entries_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id);
ALTER TABLE ladder_entries
ADD CONSTRAINT ladder_entries_event_id_fkey FOREIGN KEY (event_id) REFERENCES events (id);
ALTER TABLE scoring_presets
ADD CONSTRAINT scoring_presets_event_id_fkey FOREIGN KEY (event_id) REFERENCES events (id);
ALTER TABLE signups
ADD CONSTRAINT signups_event_id_fkey FOREIGN KEY (event_id) REFERENCES events (id);
ALTER TABLE signups
ADD CONSTRAINT signups_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id);
ALTER TABLE submissions
ADD CONSTRAINT submissions_event_id_fkey FOREIGN KEY (event_id) REFERENCES events (id);
ALTER TABLE teams
ADD CONSTRAINT team_event_id_fkey FOREIGN KEY (event_id) REFERENCES events (id);
ALTER TABLE team_users
ADD CONSTRAINT team_users_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams (id);
ALTER TABLE team_users
ADD CONSTRAINT team_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id);
-- missing unique constraints
ALTER TABLE submissions
ADD CONSTRAINT submissions_user_objective_timestamp_unique UNIQUE (user_id, objective_id, timestamp);