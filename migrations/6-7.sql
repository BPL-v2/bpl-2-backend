CREATE TABLE team_suggestions (
    id int8 NOT NULL,
    team_id int8 NOT NULL,
    is_objective bool NOT NULL,
    CONSTRAINT team_suggestions_pkey PRIMARY KEY (id, team_id, is_objective)
);