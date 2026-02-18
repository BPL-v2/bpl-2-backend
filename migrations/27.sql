ALTER TABLE guilds DROP CONSTRAINT team_guilds_pk;
ALTER TABLE guilds ADD PRIMARY KEY (id, team_id, event_id);