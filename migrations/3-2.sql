-- ALTER TABLE submissions DROP COLUMN match_id;
-- ALTER TABLE objective_matches DROP COLUMN id;
ALTER TABLE objective_matches
ADD COLUMN id bigserial NOT NULL UNIQUE;
-- set default value for match_id
ALTER TABLE objective_matches
ALTER COLUMN id
SET DEFAULT nextval('objective_matches_id_seq'::regclass);
-- set not null constraint
ALTER TABLE objective_matches
ALTER COLUMN id
SET NOT NULL;
-- create foreign key in submissions
ALTER TABLE submissions
ADD COLUMN match_id int8 NULL;
-- create foreign key constraint
ALTER TABLE submissions
ADD CONSTRAINT fk_submissions_match_id FOREIGN KEY (match_id) REFERENCES objective_matches (id);