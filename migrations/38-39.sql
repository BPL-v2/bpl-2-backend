CREATE TABLE activity (
    time TIMESTAMPTZ NOT NULL,
    user_id INT4 NOT NULL,
    event_id INT4 NOT NULL
);
CREATE INDEX idx_activity_user_event ON activity(user_id, event_id);

INSERT INTO activity (time, user_id, event_id)
select DISTINCT ON (s.character_id, s.event_id, s.xp) s.time, c.user_id, s.event_id 
from character_stats s
join characters c on s.character_id = c.id
where c.user_id is not NULL
ORDER BY s.character_id, s.event_id, s.xp, s.time;

ALTER TABLE activity ADD FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE activity ADD FOREIGN KEY(event_id) REFERENCES events(id) ON DELETE CASCADE;