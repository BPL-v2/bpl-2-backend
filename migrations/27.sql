ALTER TABLE guilds DROP CONSTRAINT guilds_pkey;
ALTER TABLE guilds ADD PRIMARY KEY (id, team_id, event_id);