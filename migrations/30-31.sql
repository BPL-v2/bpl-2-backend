ALTER TABLE atlas ALTER COLUMN user_id TYPE int2 USING user_id::int2;
ALTER TABLE atlas ALTER COLUMN event_id TYPE int2 USING event_id::int2;

ALTER TABLE change_ids ALTER COLUMN event_id TYPE int2 USING event_id::int2;

ALTER TABLE characters ALTER COLUMN user_id TYPE int2 USING user_id::int2;
ALTER TABLE characters ALTER COLUMN "level" TYPE int2 USING "level"::int2;
ALTER TABLE characters ALTER COLUMN event_id TYPE int2 USING event_id::int2;

ALTER TABLE conditions ALTER COLUMN objective_id TYPE int4 USING objective_id::int4;
ALTER TABLE conditions ALTER COLUMN id TYPE int4 USING id::int4;

ALTER TABLE objectives ALTER COLUMN id TYPE int4 USING id::int4;
ALTER TABLE objectives ALTER COLUMN parent_id TYPE int4 USING parent_id::int4;
ALTER TABLE objectives ALTER COLUMN scoring_id TYPE int4 USING scoring_id::int4;
ALTER TABLE objectives ALTER COLUMN event_id TYPE int2 USING event_id::int2;

ALTER TABLE recurring_jobs ALTER COLUMN event_id TYPE int2 USING event_id::int2;

ALTER TABLE scoring_presets ALTER COLUMN event_id TYPE int2 USING event_id::int2;
ALTER TABLE scoring_presets ALTER COLUMN id TYPE int4 USING id::int4;

ALTER TABLE signups ALTER COLUMN event_id TYPE int2 USING event_id::int2;
ALTER TABLE signups ALTER COLUMN user_id TYPE int2 USING user_id::int2;
ALTER TABLE signups ALTER COLUMN partner_id TYPE int2 USING partner_id::int2;

ALTER TABLE stash_changes ALTER COLUMN event_id TYPE int2 USING event_id::int2;

ALTER TABLE submissions ALTER COLUMN reviewer_id TYPE int4 USING reviewer_id::int4;
ALTER TABLE submissions ALTER COLUMN event_id TYPE int2 USING event_id::int2;
ALTER TABLE submissions ALTER COLUMN user_id TYPE int4 USING user_id::int4;
ALTER TABLE submissions ALTER COLUMN objective_id TYPE int4 USING objective_id::int4;
ALTER TABLE submissions ALTER COLUMN id TYPE int4 USING id::int4;

ALTER TABLE team_suggestions ALTER COLUMN id TYPE int4 USING id::int4;
ALTER TABLE team_suggestions ALTER COLUMN team_id TYPE int2 USING team_id::int2;

ALTER TABLE team_users ALTER COLUMN team_id TYPE int2 USING team_id::int2;
ALTER TABLE team_users ALTER COLUMN user_id TYPE int4 USING user_id::int4;

ALTER TABLE teams ALTER COLUMN id TYPE int4 USING id::int4;
ALTER TABLE teams ALTER COLUMN event_id TYPE int2 USING event_id::int2;

ALTER TABLE users ALTER COLUMN id TYPE int4 USING id::int4;

ALTER TABLE character_pobs ALTER COLUMN "level" TYPE int2 USING "level"::int2;

ALTER TABLE bpl2.objective_matches ALTER COLUMN event_id TYPE int2 USING event_id::int2;
ALTER TABLE bpl2.objective_matches ALTER COLUMN objective_id TYPE int4 USING objective_id::int4;
ALTER TABLE bpl2.objective_matches ALTER COLUMN user_id TYPE int4 USING user_id::int4;
ALTER TABLE bpl2.objective_matches ALTER COLUMN "number" TYPE int2 USING "number"::int2;

CREATE INDEX character_pobs_character_id_idx ON character_pobs (character_id);

ALTER TABLE bpl2.objective_matches ADD COLUMN team_id int2;

UPDATE objective_matches o
SET team_id = t.team_id
FROM bpl2.team_users t
JOIN bpl2.teams tm ON t.team_id = tm.id
JOIN bpl2.events e ON tm.event_id = e.id
WHERE o.user_id = t.user_id AND o.event_id = e.id;

ALTER TABLE bpl2.objective_matches ALTER COLUMN team_id SET NOT NULL;
ALTER TABLE bpl2.objective_matches ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE bpl2.objective_matches DROP COLUMN event_id;
ALTER TABLE bpl2.objective_matches ADD CONSTRAINT objective_matches_teams_fk FOREIGN KEY (team_id) REFERENCES bpl2.teams(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE bpl2.submissions ADD team_id int2 NULL;
UPDATE bpl2.submissions s
SET team_id = t.team_id
FROM bpl2.team_users t
JOIN bpl2.teams tm ON t.team_id = tm.id
JOIN bpl2.events e ON tm.event_id = e.id
WHERE s.user_id = t.user_id AND s.event_id = e.id;
ALTER TABLE bpl2.submissions ALTER COLUMN team_id SET NOT NULL;
ALTER TABLE bpl2.submissions DROP CONSTRAINT submissions_event_id_fkey;
ALTER TABLE bpl2.submissions DROP COLUMN event_id;
ALTER TABLE bpl2.submissions ADD CONSTRAINT submissions_teams_fk FOREIGN KEY (team_id) REFERENCES bpl2.teams(id) ON DELETE CASCADE ON UPDATE CASCADE;
