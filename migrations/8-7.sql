-- ALTER TABLE signups drop column expected_play_time;
-- ALTER TABLE signups
-- add column expected_play_time INTEGER NOT NULL DEFAULT 0;
ALTER TABLE signups drop column expected_play_time;
ALTER TABLE signups
add column expected_play_time text NOT NULL DEFAULT 'VERY_LOW';