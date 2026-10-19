-- +goose Up
CREATE TABLE deaths (
	"time" timestamptz NOT NULL,
	character_id text NOT NULL,
	CONSTRAINT deaths_character_id_fkey FOREIGN KEY (character_id) REFERENCES "characters"(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX idx_deaths_character_time ON deaths USING btree (character_id, "time");

-- +goose Down
DROP TABLE IF EXISTS deaths;
