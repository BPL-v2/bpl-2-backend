CREATE TABLE character_stats (
	"time" timestamptz NOT NULL,
	event_id int2 NOT NULL,
	character_id text NOT NULL,
	dps int8 NOT NULL,
	ehp int4 NOT NULL,
	phys_max_hit int4 NOT NULL,
	ele_max_hit int4 NOT NULL,
	hp int4 NOT NULL,
	mana int4 NOT NULL,
	es int4 NOT NULL,
	armour int4 NOT NULL,
	evasion int4 NOT NULL,
	xp int8 NOT NULL,
	movement_speed int4 DEFAULT 0 NOT NULL
);
CREATE INDEX character_stats_event_id_idx ON character_stats USING btree (event_id DESC);
CREATE INDEX character_stats_time_idx ON character_stats USING btree ("time" DESC);


create trigger ts_insert_blocker before
insert
    on
    character_stats for each row execute function _timescaledb_functions.insert_blocker();



CREATE TABLE client_credentials (
	"name" text NOT NULL,
	access_token text NULL,
	expiry timestamptz NULL,
	CONSTRAINT client_credentials_pkey PRIMARY KEY (name)
);



CREATE TABLE events (
	id bigserial NOT NULL,
	"name" text NOT NULL,
	is_current bool NOT NULL,
	game_version text NOT NULL,
	max_size int8 NOT NULL,
	application_start_time timestamptz NOT NULL,
	event_start_time timestamptz NOT NULL,
	event_end_time timestamptz NOT NULL,
	public bool NOT NULL,
	"locked" bool NOT NULL,
	waitlist_size int4 DEFAULT 0 NOT NULL,
	application_end_time timestamptz DEFAULT now() NOT NULL,
	patch varchar NULL,
	CONSTRAINT events_pkey PRIMARY KEY (id)
);



CREATE TABLE kafka_consumers (
	event_id bigserial NOT NULL,
	group_id int8 NOT NULL,
	CONSTRAINT kafka_consumers_pkey PRIMARY KEY (event_id)
);


CREATE TABLE users (
	id serial4 NOT NULL,
	display_name text NOT NULL,
	permissions _text DEFAULT '{}'::text[] NOT NULL,
	CONSTRAINT users_pkey PRIMARY KEY (id)
);


CREATE TABLE activity (
	"time" timestamptz NOT NULL,
	user_id int4 NOT NULL,
	event_id int2 NOT NULL,
	CONSTRAINT activity_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
	CONSTRAINT activity_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_activity_user_event ON activity USING btree (user_id, event_id);

CREATE TABLE atlas_trees (
	event_id int2 NOT NULL,
	user_id int4 NOT NULL,
	"index" int2 NOT NULL,
	nodes _int2 NOT NULL,
	"timestamp" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT atlas_trees_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT atlas_trees_users_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX atlas_trees_event_id_idx ON atlas_trees USING btree (event_id);
CREATE INDEX atlas_trees_user_id_idx ON atlas_trees USING btree (user_id);

CREATE TABLE cached_data (
	"key" int2 NOT NULL,
	event_id int2 NOT NULL,
	"data" bytea NOT NULL,
	"timestamp" timestamptz NOT NULL,
	CONSTRAINT cached_data_pk PRIMARY KEY (key, event_id),
	CONSTRAINT cached_data_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE change_ids (
	current_change_id text NOT NULL,
	next_change_id text NOT NULL,
	event_id int2 NOT NULL,
	"timestamp" timestamptz NULL,
	CONSTRAINT change_ids_pkey PRIMARY KEY (event_id),
	CONSTRAINT change_ids_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);


CREATE TABLE "characters" (
	id text DEFAULT nextval('characters2_id_seq'::regclass) NOT NULL,
	user_id int2 NULL,
	event_id int2 NOT NULL,
	"name" varchar(255) NOT NULL,
	"level" int2 NOT NULL,
	main_skill varchar(255) NOT NULL,
	ascendancy varchar(255) NOT NULL,
	ascendancy_points int4 NOT NULL,
	atlas_points int4 NOT NULL,
	pantheon bool NOT NULL,
	old_account_name varchar(255) NULL,
	CONSTRAINT characters2_pkey PRIMARY KEY (id),
	CONSTRAINT characters2_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT characters2_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX characters_old_account_name_idx ON characters USING btree (old_account_name);
CREATE INDEX idx_characters2_event_id ON characters USING btree (event_id);
CREATE INDEX idx_characters2_user_id ON characters USING btree (user_id);

CREATE TABLE guild_stash_changelogs (
	id int8 NOT NULL,
	"timestamp" timestamptz NOT NULL,
	guild_id int4 NOT NULL,
	event_id int2 NOT NULL,
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
CREATE INDEX guild_stash_changelogs_account_name_idx ON guild_stash_changelogs USING btree (account_name);
CREATE INDEX guild_stash_changelogs_event_id_idx ON guild_stash_changelogs USING btree (event_id);
CREATE INDEX guild_stash_changelogs_guild_id_idx ON guild_stash_changelogs USING btree (guild_id);
CREATE INDEX guild_stash_changelogs_item_name_idx ON guild_stash_changelogs USING btree (item_name);
CREATE INDEX guild_stash_changelogs_stash_name_idx ON guild_stash_changelogs USING btree (stash_name);
CREATE INDEX guild_stash_changelogs_timestamp_idx ON guild_stash_changelogs USING btree ("timestamp");

CREATE TABLE ladder_entries (
	user_id int4 NULL,
	account text NOT NULL,
	"character" text NOT NULL,
	"class" text NOT NULL,
	"level" int2 NOT NULL,
	delve int4 NOT NULL,
	experience int8 NOT NULL,
	event_id int2 NOT NULL,
	"rank" int4 NOT NULL,
	twitch_account text NULL,
	CONSTRAINT ladder_entries_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT ladder_entries_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE INDEX idx_bpl2_ladder_entries_event_id ON ladder_entries USING btree (event_id);
CREATE INDEX idx_bpl2_ladder_entries_user_id ON ladder_entries USING btree (user_id);

CREATE TABLE oauths (
	user_id int4 NOT NULL,
	provider text NOT NULL,
	access_token text NOT NULL,
	refresh_token text NULL,
	expiry timestamptz NOT NULL,
	"name" text NOT NULL,
	account_id text NOT NULL,
	CONSTRAINT oauths_pkey PRIMARY KEY (user_id, provider),
	CONSTRAINT oauths_unique UNIQUE (account_id),
	CONSTRAINT fk_users_oauth_accounts FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX idx_oauths_provider_account_id ON oauths USING btree (provider, account_id);
CREATE INDEX idx_oauths_provider_account_name ON oauths USING btree (provider, name);
CREATE INDEX idx_oauths_user_id ON oauths USING btree (user_id);

CREATE TABLE recurring_jobs (
	job_type text NOT NULL,
	event_id int2 NOT NULL,
	sleep_after_each_run_seconds int8 NOT NULL,
	end_date timestamptz NOT NULL,
	CONSTRAINT uni_bpl2_recurring_jobs_job_type PRIMARY KEY (job_type),
	CONSTRAINT recurring_jobs_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE scoring_presets (
	id serial4 NOT NULL,
	event_id int2 NOT NULL,
	"name" text NOT NULL,
	description text NOT NULL,
	points _numeric NOT NULL,
	"scoring_method" text NOT NULL,
	"type" text NOT NULL,
	point_cap int4 DEFAULT 0 NOT NULL,
	CONSTRAINT scoring_presets_pkey PRIMARY KEY (id),
	CONSTRAINT scoring_presets_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX scoring_presets_event_id_idx ON scoring_presets USING btree (event_id);

CREATE TABLE signups (
	event_id int2 NOT NULL,
	user_id int2 NOT NULL,
	"timestamp" timestamptz NOT NULL,
	expected_play_time int4 DEFAULT 0 NOT NULL,
	needs_help bool DEFAULT false NOT NULL,
	wants_to_help bool DEFAULT false NOT NULL,
	partner_id int2 NULL,
	actual_play_time int4 DEFAULT 0 NOT NULL,
	extra text NULL,
	CONSTRAINT signups_pkey PRIMARY KEY (event_id, user_id),
	CONSTRAINT fk_signup_partner FOREIGN KEY (partner_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
	CONSTRAINT signups_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT signups_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX idx_signup_partner ON signups USING btree (partner_id);
CREATE INDEX signups_event_id_idx ON signups USING btree (event_id);
CREATE INDEX signups_user_id_idx ON signups USING btree (user_id);

CREATE TABLE stash_changes (
	stash_id text NOT NULL,
	event_id int2 NOT NULL,
	"timestamp" timestamptz NULL,
	id serial4 NOT NULL,
	CONSTRAINT stash_changes_pkey PRIMARY KEY (id),
	CONSTRAINT stash_changes_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX stash_changes_event_id_idx ON stash_changes USING btree (event_id);

CREATE TABLE teams (
	id serial4 NOT NULL,
	"name" text NOT NULL,
	allowed_classes _text NOT NULL,
	event_id int2 NOT NULL,
	color text NOT NULL,
	abbreviation text DEFAULT ''::text NOT NULL,
	CONSTRAINT teams_pkey PRIMARY KEY (id),
	CONSTRAINT fk_bpl2_events_teams FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX team_event_id_idx ON teams USING btree (event_id);

CREATE TABLE character_pobs (
	id bigserial NOT NULL,
	character_id text NOT NULL,
	export bytea NOT NULL,
	"level" int2 NOT NULL,
	ascendancy text NOT NULL,
	main_skill text NOT NULL,
	"timestamp" timestamptz DEFAULT now() NOT NULL,
	CONSTRAINT character_pobs_pkey PRIMARY KEY (id),
	CONSTRAINT character_pobs_character_id_fkey FOREIGN KEY (character_id) REFERENCES "characters"(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX character_pobs_character_id_idx ON character_pobs USING btree (character_id);

CREATE TABLE guild_stash_tabs (
	id text NOT NULL,
	event_id int2 NOT NULL,
	team_id int2 NOT NULL,
	"name" text NOT NULL,
	"type" text NOT NULL,
	"index" int8 NULL,
	color text NULL,
	owner_id int8 NOT NULL,
	parent_id text NULL,
	parent_event_id int2 NULL,
	raw text DEFAULT ''::text NOT NULL,
	fetch_enabled bool NOT NULL,
	user_ids _int4 NOT NULL,
	last_fetch timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT guild_stash_tabs_pkey PRIMARY KEY (id, event_id),
	CONSTRAINT fk_bpl2_guild_stash_tabs_event FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT fk_bpl2_guild_stash_tabs_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT fk_bpl2_guild_stash_tabs_parent FOREIGN KEY (parent_id,parent_event_id) REFERENCES guild_stash_tabs(id,event_id) ON DELETE SET NULL ON UPDATE CASCADE,
	CONSTRAINT fk_bpl2_guild_stash_tabs_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX guild_stash_tab_event_idx ON guild_stash_tabs USING btree (event_id);
CREATE INDEX guild_stash_tab_team_idx ON guild_stash_tabs USING btree (team_id);

CREATE TABLE guilds (
	team_id int2 NOT NULL,
	id int4 NOT NULL,
	"name" varchar NULL,
	tag varchar NULL,
	CONSTRAINT team_guilds_pk PRIMARY KEY (team_id, id),
	CONSTRAINT team_guilds_teams_fk FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE objectives (
	id serial4 NOT NULL,
	"name" text NOT NULL,
	extra text NULL,
	required_amount int8 NOT NULL,
	parent_id int4 NULL,
	"objective_type" text NOT NULL,
	"number_field" text NOT NULL,
	aggregation text NOT NULL,
	valid_from timestamptz NULL,
	valid_to timestamptz NULL,
	scoring_id int4 NULL,
	sync_status text NULL,
	event_id int2 NULL,
	hide_progress bool DEFAULT false NOT NULL,
	CONSTRAINT objectives_pkey PRIMARY KEY (id),
	CONSTRAINT fk_bpl2_objectives_scoring_preset FOREIGN KEY (scoring_id) REFERENCES scoring_presets(id) ON DELETE SET NULL ON UPDATE CASCADE,
	CONSTRAINT objectives_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT objectives_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX objectives_event_id_idx ON objectives USING btree (event_id);

CREATE TABLE submissions (
	id serial4 NOT NULL,
	objective_id int4 NOT NULL,
	"timestamp" timestamptz NOT NULL,
	"number" int8 NOT NULL,
	user_id int4 NOT NULL,
	proof text NOT NULL,
	"comment" text NOT NULL,
	"approval_status" text NOT NULL,
	review_comment text NULL,
	reviewer_id int4 NULL,
	team_id int2 NOT NULL,
	CONSTRAINT submissions_pkey PRIMARY KEY (id),
	CONSTRAINT submissions_user_objective_timestamp_unique UNIQUE (user_id, objective_id, "timestamp"),
	CONSTRAINT fk_bpl2_submissions_objective FOREIGN KEY (objective_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT fk_bpl2_submissions_reviewer FOREIGN KEY (reviewer_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE SET NULL,
	CONSTRAINT fk_bpl2_submissions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT submissions_teams_fk FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE team_suggestions (
	id int4 NOT NULL,
	team_id int2 NOT NULL,
	extra text DEFAULT ''::text NOT NULL,
	CONSTRAINT team_suggestions_pkey PRIMARY KEY (id, team_id),
	CONSTRAINT team_suggestions_teams_fk FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX team_suggestions_team_id_idx ON team_suggestions USING btree (team_id);

CREATE TABLE team_users (
	team_id int2 NOT NULL,
	user_id int4 NOT NULL,
	is_team_lead bool DEFAULT false NOT NULL,
	CONSTRAINT team_users_pkey PRIMARY KEY (team_id, user_id),
	CONSTRAINT team_users_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT team_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX idx_bpl2_team_users_team_id ON team_users USING btree (team_id);
CREATE INDEX idx_bpl2_team_users_user_id ON team_users USING btree (user_id);

CREATE TABLE conditions (
	id serial4 NOT NULL,
	objective_id int4 NOT NULL,
	field text NOT NULL,
	"operator" text NOT NULL,
	value text NOT NULL,
	CONSTRAINT conditions_pkey PRIMARY KEY (id),
	CONSTRAINT fk_bpl2_objectives_conditions FOREIGN KEY (objective_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE objective_matches (
	objective_id int4 NOT NULL,
	"timestamp" timestamptz NOT NULL,
	"number" int2 NOT NULL,
	user_id int4 NULL,
	stash_change_id int4 NULL,
	team_id int2 NOT NULL,
	CONSTRAINT objective_matches_objectives_fk FOREIGN KEY (objective_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT objective_matches_stash_changes_fk FOREIGN KEY (stash_change_id) REFERENCES stash_changes(id) ON DELETE SET NULL ON UPDATE CASCADE,
	CONSTRAINT objective_matches_teams_fk FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT objective_matches_users_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX objective_matches_objective_id_idx ON objective_matches USING btree (objective_id);

CREATE TABLE objective_validations (
	objective_id int4 NOT NULL,
	item jsonb NOT NULL,
	"timestamp" timestamptz NOT NULL,
	CONSTRAINT objective_validations_pkey PRIMARY KEY (objective_id),
	CONSTRAINT objective_validations_objectives_fk FOREIGN KEY (objective_id) REFERENCES objectives(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX idx_objective_validations_objective_id ON objective_validations USING btree (objective_id);