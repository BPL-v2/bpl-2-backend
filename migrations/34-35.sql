CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE guild_stash_changelogs (
	id int8 NOT NULL,
	timestamp timestamptz NOT NULL,
	guild_id int4 NOT NULL,
	event_id int4 NOT NULL,
	"action" int2 NOT NULL,
	"number" int4 NOT NULL,
	x int2 NOT NULL,
	y int2 NOT NULL,
	stash_name varchar NULL,
	account_name varchar NOT NULL,
	item_name varchar NOT NULL,
	CONSTRAINT guild_stash_changelogs_pk PRIMARY KEY (id),
	CONSTRAINT guild_stash_changelogs_events_fk FOREIGN KEY (event_id) REFERENCES events(id)
);
CREATE INDEX guild_stash_changelogs_timestamp_idx ON guild_stash_changelogs (timestamp);
CREATE INDEX guild_stash_changelogs_item_name_idx ON guild_stash_changelogs (item_name);
CREATE INDEX guild_stash_changelogs_account_name_idx ON guild_stash_changelogs (account_name);
CREATE INDEX guild_stash_changelogs_event_id_idx ON guild_stash_changelogs (event_id);
CREATE INDEX guild_stash_changelogs_guild_id_idx ON guild_stash_changelogs (guild_id);
CREATE INDEX guild_stash_changelogs_stash_name_idx ON guild_stash_changelogs (stash_name);

CREATE TABLE team_guilds (
	team_id int4 NOT NULL,
	guild_id int4 NOT NULL,
	CONSTRAINT team_guilds_pk PRIMARY KEY (team_id, guild_id),
	CONSTRAINT team_guilds_teams_fk FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE
);