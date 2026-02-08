ALTER TABLE oauths
ADD COLUMN refresh_expiry TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00';

UPDATE oauths
SET refresh_expiry = expiry + INTERVAL '1 day'