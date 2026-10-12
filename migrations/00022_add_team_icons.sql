-- +goose Up
ALTER TABLE teams ADD COLUMN icon_mime_type text NULL;
ALTER TABLE teams ADD COLUMN icon_updated_at timestamptz NULL;

CREATE TABLE team_icons (
    team_id int4 PRIMARY KEY REFERENCES teams(id) ON DELETE CASCADE,
    icon bytea NOT NULL,
    mime_type text NOT NULL
);

-- +goose Down
DROP TABLE team_icons;
ALTER TABLE teams DROP COLUMN icon_updated_at;
ALTER TABLE teams DROP COLUMN icon_mime_type;
