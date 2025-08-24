ALTER TABLE team_guilds ADD COLUMN IF NOT EXISTS "name" varchar NULL;
ALTER TABLE team_guilds ADD COLUMN IF NOT EXISTS tag varchar NULL;

ALTER TABLE team_guilds RENAME TO guilds;
ALTER TABLE guilds RENAME COLUMN guild_id TO id;