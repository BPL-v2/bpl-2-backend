CREATE SCHEMA IF NOT EXISTS bpl2;
CREATE TABLE bpl2.client_credentials (
    "name" text NOT NULL,
    access_token text NULL,
    expiry timestamptz NULL,
    CONSTRAINT client_credentials_pkey PRIMARY KEY (name)
);
CREATE TABLE bpl2.kafka_consumers (
    event_id bigserial NOT NULL,
    group_id int8 NOT NULL,
    CONSTRAINT kafka_consumers_pkey PRIMARY KEY (event_id)
);
CREATE TABLE bpl2.ladder_entries (
    account text NOT NULL,
    "character" text NOT NULL,
    "class" text NOT NULL,
    "level" int8 NOT NULL,
    delve int8 NOT NULL,
    experience int8 NOT NULL,
    event_id int8 NOT NULL,
    user_id int8 NOT NULL,
    "rank" int8 NOT NULL
);
CREATE INDEX idx_bpl2_ladder_entries_event_id ON bpl2.ladder_entries USING btree (event_id);
CREATE INDEX idx_bpl2_ladder_entries_user_id ON bpl2.ladder_entries USING btree (user_id);
CREATE TABLE bpl2.objective_matches (
    objective_id int8 NOT NULL,
    "timestamp" timestamptz NOT NULL,
    "number" int8 NOT NULL,
    user_id int8 NOT NULL,
    event_id int8 NOT NULL,
    stash_change_id int8 NULL
);
CREATE INDEX obj_match_event ON bpl2.objective_matches USING btree (event_id);
CREATE INDEX obj_match_obj ON bpl2.objective_matches USING btree (objective_id);
CREATE INDEX obj_match_obj_user ON bpl2.objective_matches USING btree (objective_id, user_id);
CREATE INDEX obj_match_stash_change ON bpl2.objective_matches USING btree (stash_change_id);
CREATE INDEX obj_match_user ON bpl2.objective_matches USING btree (user_id);
CREATE TABLE bpl2.recurring_jobs (
    job_type text NOT NULL,
    event_id int8 NOT NULL,
    sleep_after_each_run_seconds int8 NOT NULL,
    end_date timestamptz NOT NULL,
    CONSTRAINT uni_bpl2_recurring_jobs_job_type PRIMARY KEY (job_type)
);
CREATE TABLE bpl2.scoring_presets (
    id bigserial NOT NULL,
    event_id int8 NOT NULL,
    "name" text NOT NULL,
    description text NOT NULL,
    points _numeric NOT NULL,
    "scoring_method" text NOT NULL,
    "type" text NOT NULL,
    CONSTRAINT scoring_presets_pkey PRIMARY KEY (id)
);
CREATE TABLE bpl2.stash_changes (
    next_change_id text NOT NULL,
    event_id int8 NOT NULL,
    "timestamp" timestamptz NULL,
    id bigserial NOT NULL,
    stash_id text NOT NULL
);
CREATE INDEX idx_bpl2_stash_changes_event_id ON bpl2.stash_changes USING btree (event_id);
CREATE INDEX stash_changes_event_id_idx ON bpl2.stash_changes USING btree (event_id);
CREATE TABLE bpl2.team_users (
    team_id int8 NOT NULL,
    user_id int8 NOT NULL,
    is_team_lead bool DEFAULT false NOT NULL,
    CONSTRAINT team_users_pkey PRIMARY KEY (team_id, user_id)
);
CREATE INDEX idx_bpl2_team_users_team_id ON bpl2.team_users USING btree (team_id);
CREATE INDEX idx_bpl2_team_users_user_id ON bpl2.team_users USING btree (user_id);
CREATE TABLE bpl2.users (
    id bigserial NOT NULL,
    display_name text NOT NULL,
    permissions _text DEFAULT '{}'::text [] NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);
CREATE TABLE bpl2.oauths (
    user_id int8 NOT NULL,
    provider text NOT NULL,
    access_token text NOT NULL,
    refresh_token text NULL,
    expiry timestamptz NOT NULL,
    "name" text NOT NULL,
    account_id text NOT NULL,
    CONSTRAINT oauths_pkey PRIMARY KEY (user_id, provider),
    CONSTRAINT fk_bpl2_users_oauth_accounts FOREIGN KEY (user_id) REFERENCES bpl2.users(id)
);
CREATE TABLE bpl2.scoring_categories (
    id bigserial NOT NULL,
    "name" text NOT NULL,
    parent_id int8 NULL,
    scoring_id int8 NULL,
    CONSTRAINT scoring_categories_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bpl2_scoring_categories_scoring_preset FOREIGN KEY (scoring_id) REFERENCES bpl2.scoring_presets(id),
    CONSTRAINT fk_bpl2_scoring_categories_sub_categories FOREIGN KEY (parent_id) REFERENCES bpl2.scoring_categories(id) ON DELETE CASCADE
);
CREATE TABLE bpl2.signups (
    id bigserial NOT NULL,
    event_id int8 NOT NULL,
    user_id int8 NOT NULL,
    "timestamp" timestamptz NOT NULL,
    expected_play_time text NOT NULL,
    CONSTRAINT signups_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bpl2_signups_user FOREIGN KEY (user_id) REFERENCES bpl2.users(id)
);
CREATE TABLE bpl2.events (
    id bigserial NOT NULL,
    "name" text NOT NULL,
    scoring_category_id int8 NOT NULL,
    is_current bool NOT NULL,
    max_size int8 NOT NULL,
    application_start_time timestamptz NULL,
    event_start_time timestamptz NULL,
    event_end_time timestamptz NULL,
    game_version text NOT NULL,
    CONSTRAINT events_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bpl2_events_scoring_category FOREIGN KEY (scoring_category_id) REFERENCES bpl2.scoring_categories(id) ON DELETE CASCADE
);
CREATE TABLE bpl2.objectives (
    id bigserial NOT NULL,
    "name" text NOT NULL,
    extra text NULL,
    required_amount int8 NOT NULL,
    category_id int8 NOT NULL,
    "objective_type" text NOT NULL,
    "number_field" text NOT NULL,
    aggregation text NOT NULL,
    valid_from timestamptz NULL,
    valid_to timestamptz NULL,
    scoring_id int8 NULL,
    sync_status text NULL,
    CONSTRAINT objectives_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bpl2_objectives_scoring_preset FOREIGN KEY (scoring_id) REFERENCES bpl2.scoring_presets(id),
    CONSTRAINT fk_bpl2_scoring_categories_objectives FOREIGN KEY (category_id) REFERENCES bpl2.scoring_categories(id) ON DELETE CASCADE
);
CREATE TABLE bpl2.submissions (
    id bigserial NOT NULL,
    objective_id int8 NOT NULL,
    "timestamp" timestamptz NOT NULL,
    "number" int8 NOT NULL,
    user_id int8 NOT NULL,
    proof text NOT NULL,
    "comment" text NOT NULL,
    "approval_status" text NOT NULL,
    review_comment text NULL,
    reviewer_id int8 NULL,
    event_id int8 NOT NULL,
    CONSTRAINT submissions_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bpl2_submissions_objective FOREIGN KEY (objective_id) REFERENCES bpl2.objectives(id) ON DELETE CASCADE,
    CONSTRAINT fk_bpl2_submissions_reviewer FOREIGN KEY (reviewer_id) REFERENCES bpl2.users(id) ON DELETE CASCADE,
    CONSTRAINT fk_bpl2_submissions_user FOREIGN KEY (user_id) REFERENCES bpl2.users(id) ON DELETE CASCADE
);
CREATE TABLE bpl2.teams (
    id bigserial NOT NULL,
    "name" text NOT NULL,
    allowed_classes _text NOT NULL,
    event_id int8 NOT NULL,
    CONSTRAINT teams_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bpl2_events_teams FOREIGN KEY (event_id) REFERENCES bpl2.events(id) ON DELETE CASCADE
);
CREATE TABLE bpl2.conditions (
    id bigserial NOT NULL,
    objective_id int8 NOT NULL,
    field text NOT NULL,
    "operator" text NOT NULL,
    value text NOT NULL,
    CONSTRAINT conditions_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bpl2_objectives_conditions FOREIGN KEY (objective_id) REFERENCES bpl2.objectives(id) ON DELETE CASCADE
);