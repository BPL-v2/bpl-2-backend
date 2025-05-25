CREATE TABLE scoring_categories (
    id bigserial NOT NULL,
    "name" text NOT NULL,
    parent_id int8 NULL,
    scoring_id int8 NULL,
    event_id int8 NOT NULL,
    CONSTRAINT scoring_categories_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bpl2_scoring_categories_scoring_preset FOREIGN KEY (scoring_id) REFERENCES scoring_presets(id),
    CONSTRAINT fk_bpl2_scoring_categories_sub_categories FOREIGN KEY (parent_id) REFERENCES scoring_categories(id) ON DELETE CASCADE
);
ALTER TABLE team_suggestions
ADD COLUMN is_objective bool DEFAULT false NOT NULL;