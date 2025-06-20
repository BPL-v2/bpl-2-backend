ALTER TABLE team_users DROP CONSTRAINT IF EXISTS team_users_team_id_fkey;
ALTER TABLE team_users DROP CONSTRAINT IF EXISTS team_users_user_id_fkey;
ALTER TABLE team_users
ADD CONSTRAINT team_users_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE team_users
ADD CONSTRAINT team_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;