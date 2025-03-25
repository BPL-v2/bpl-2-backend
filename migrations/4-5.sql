-- "characters" definition
-- Drop table
-- DROP TABLE "characters";
CREATE TABLE "characters" (
    user_id int8 NOT NULL,
    event_id int8 NOT NULL,
    "level" int8 NOT NULL,
    ascendancy_points int8 NOT NULL,
    "timestamp" timestamptz NOT NULL,
    pantheon bool NOT NULL,
    "name" text NOT NULL,
    main_skill text NOT NULL,
    ascendancy text NOT NULL
);
CREATE INDEX idx_characters_event_id ON characters USING btree (event_id);
CREATE INDEX idx_characters_timestamp ON characters USING btree ("timestamp");
CREATE INDEX idx_characters_user_id ON characters USING btree (user_id);
-- "characters" foreign keys
ALTER TABLE "characters"
ADD CONSTRAINT fk_characters_event FOREIGN KEY (event_id) REFERENCES events(id);
ALTER TABLE "characters"
ADD CONSTRAINT fk_characters_user FOREIGN KEY (user_id) REFERENCES users(id);
-- atlas definition
-- Drop table
-- DROP TABLE atlas;
CREATE TABLE atlas (
    user_id int8 NOT NULL,
    event_id int8 NOT NULL,
    "index" int8 NOT NULL,
    tree1 _int4 NOT NULL,
    tree2 _int4 NOT NULL,
    tree3 _int4 NOT NULL,
    CONSTRAINT atlas_pkey PRIMARY KEY (user_id, event_id)
);
CREATE INDEX idx_atlas_event_id ON atlas USING btree (event_id);
-- atlas foreign keys
ALTER TABLE atlas
ADD CONSTRAINT fk_atlas_event FOREIGN KEY (event_id) REFERENCES events(id);
ALTER TABLE atlas
ADD CONSTRAINT fk_atlas_user FOREIGN KEY (user_id) REFERENCES users(id);