ALTER TABLE events
add column application_end_time timestamptz NOT NULL DEFAULT now();