CREATE TABLE guild_stash_tabs (
    id text NOT NULL,
    event_id int8 NOT NULL,
    team_id int8 NOT NULL,
    "name" text NOT NULL,
    "type" text NOT NULL,
    "index" int8 NULL,
    color text NULL,
    owner_id int8 NOT NULL,
    parent_id text NULL,
    parent_event_id int8 NULL,
    items text DEFAULT ''::text NOT NULL,
    fetch_enabled bool NOT NULL,
    user_ids _int4 NOT NULL,
    last_fetch timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT guild_stash_tabs_pkey PRIMARY KEY (id, event_id),
    CONSTRAINT fk_bpl2_guild_stash_tabs_event FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_bpl2_guild_stash_tabs_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_bpl2_guild_stash_tabs_parent FOREIGN KEY (parent_id, parent_event_id) REFERENCES guild_stash_tabs(id, event_id) ON DELETE
    SET NULL ON UPDATE CASCADE,
        CONSTRAINT fk_bpl2_guild_stash_tabs_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX guild_stash_tab_event_idx ON guild_stash_tabs USING btree (event_id);
CREATE INDEX guild_stash_tab_team_idx ON guild_stash_tabs USING btree (team_id);