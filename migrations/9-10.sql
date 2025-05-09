ALTER TABLE signups
add column needs_help BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE signups
add column wants_to_help BOOLEAN NOT NULL DEFAULT FALSE;