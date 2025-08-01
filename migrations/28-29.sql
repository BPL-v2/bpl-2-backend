ALTER TABLE atlas DROP CONSTRAINT IF EXISTS fk_atlas_event;
ALTER TABLE atlas ADD CONSTRAINT fk_atlas_event FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE atlas DROP CONSTRAINT IF EXISTS fk_atlas_user;
ALTER TABLE atlas ADD CONSTRAINT fk_atlas_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE character_pobs DROP CONSTRAINT IF EXISTS character_pobs_character_id_fkey;
ALTER TABLE character_pobs ADD CONSTRAINT character_pobs_character_id_fkey FOREIGN KEY (character_id) REFERENCES "characters"(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE character_stats DROP CONSTRAINT IF EXISTS character_stats_event_id_fkey;
ALTER TABLE character_stats ADD CONSTRAINT character_stats_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "characters" DROP CONSTRAINT IF EXISTS characters2_event_id_fkey;
ALTER TABLE "characters" ADD CONSTRAINT characters2_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "characters" DROP CONSTRAINT IF EXISTS characters2_user_id_fkey;
ALTER TABLE "characters" ADD CONSTRAINT characters2_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE conditions DROP CONSTRAINT IF EXISTS fk_bpl2_objectives_conditions;
ALTER TABLE conditions ADD CONSTRAINT fk_bpl2_objectives_conditions FOREIGN KEY (objective_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE ladder_entries DROP CONSTRAINT IF EXISTS ladder_entries_event_id_fkey;
ALTER TABLE ladder_entries ADD CONSTRAINT ladder_entries_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE ladder_entries DROP CONSTRAINT IF EXISTS ladder_entries_user_id_fkey;
ALTER TABLE ladder_entries ADD CONSTRAINT ladder_entries_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE oauths DROP CONSTRAINT IF EXISTS fk_users_oauth_accounts;
ALTER TABLE oauths ADD CONSTRAINT fk_users_oauth_accounts FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE objective_matches DROP CONSTRAINT IF EXISTS objective_matches_event_id_fkey;
ALTER TABLE objective_matches ADD CONSTRAINT objective_matches_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE objective_matches DROP CONSTRAINT IF EXISTS objective_matches_objectives_fk;
ALTER TABLE objective_matches ADD CONSTRAINT objective_matches_objectives_fk FOREIGN KEY (objective_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE objective_matches DROP CONSTRAINT IF EXISTS objective_matches_stash_change_id_fkey;
ALTER TABLE objective_matches ADD CONSTRAINT objective_matches_stash_change_id_fkey FOREIGN KEY (stash_change_id) REFERENCES stash_changes(id) ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE objective_matches DROP CONSTRAINT IF EXISTS objective_matches_users_fk;
ALTER TABLE objective_matches ADD CONSTRAINT objective_matches_users_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE objectives DROP CONSTRAINT IF EXISTS fk_bpl2_objectives_scoring_preset;
ALTER TABLE objectives ADD CONSTRAINT fk_bpl2_objectives_scoring_preset FOREIGN KEY (scoring_id) REFERENCES scoring_presets(id) ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE objectives DROP CONSTRAINT IF EXISTS objectives_event_id_fkey;
ALTER TABLE objectives ADD CONSTRAINT objectives_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE objectives DROP CONSTRAINT IF EXISTS objectives_parent_id_fkey;
ALTER TABLE objectives ADD CONSTRAINT objectives_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE scoring_presets DROP CONSTRAINT IF EXISTS scoring_presets_event_id_fkey;
ALTER TABLE scoring_presets ADD CONSTRAINT scoring_presets_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE signups DROP CONSTRAINT IF EXISTS fk_signup_partner;
ALTER TABLE signups ADD CONSTRAINT fk_signup_partner FOREIGN KEY (partner_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE signups DROP CONSTRAINT IF EXISTS signups_event_id_fkey;
ALTER TABLE signups ADD CONSTRAINT signups_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE signups DROP CONSTRAINT IF EXISTS signups_user_id_fkey;
ALTER TABLE signups ADD CONSTRAINT signups_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE submissions DROP CONSTRAINT IF EXISTS fk_bpl2_submissions_objective;
ALTER TABLE submissions ADD CONSTRAINT fk_bpl2_submissions_objective FOREIGN KEY (objective_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS fk_bpl2_submissions_reviewer;
ALTER TABLE submissions ADD CONSTRAINT fk_bpl2_submissions_reviewer FOREIGN KEY (reviewer_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE SET NULL;
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS fk_bpl2_submissions_user;
ALTER TABLE submissions ADD CONSTRAINT fk_bpl2_submissions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_event_id_fkey;
ALTER TABLE submissions ADD CONSTRAINT submissions_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE team_users DROP CONSTRAINT IF EXISTS team_users_team_id_fkey;
ALTER TABLE team_users ADD CONSTRAINT team_users_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE team_users DROP CONSTRAINT IF EXISTS team_users_user_id_fkey;
ALTER TABLE team_users ADD CONSTRAINT team_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE teams DROP CONSTRAINT IF EXISTS team_event_id_fkey;
ALTER TABLE teams DROP CONSTRAINT IF EXISTS fk_bpl2_events_teams;
ALTER TABLE teams ADD CONSTRAINT fk_bpl2_events_teams FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE change_ids ADD CONSTRAINT change_ids_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE recurring_jobs ADD CONSTRAINT recurring_jobs_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE stash_changes ADD CONSTRAINT stash_changes_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE team_suggestions ADD CONSTRAINT team_suggestions_teams_fk FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE;

DROP TABLE IF EXISTS characters_old;
