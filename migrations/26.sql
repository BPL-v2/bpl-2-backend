ALTER TABLE guilds ADD COLUMN event_id INT;

UPDATE guilds g
SET event_id = (SELECT t.event_id
                FROM teams t
                WHERE t.id = g.team_id);

ALTER TABLE guilds
ADD CONSTRAINT fk_guilds_event_id
FOREIGN KEY (event_id) REFERENCES events(id)
ON DELETE CASCADE;

ALTER TABLE guilds
ALTER COLUMN event_id SET NOT NULL;
