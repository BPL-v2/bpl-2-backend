ALTER TABLE signups drop column expected_play_time;
ALTER TABLE signups
add column expected_play_time INTEGER NOT NULL DEFAULT 0;