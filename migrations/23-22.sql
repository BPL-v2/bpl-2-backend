ALTER TABLE signups ADD CONSTRAINT unique_user_event UNIQUE (event_id, user_id);
ALTER TABLE signups DROP CONSTRAINT IF EXISTS signups_pkey;
ALTER TABLE signups ADD COLUMN id BIGSERIAL PRIMARY KEY;
ALTER TABLE signups DROP COLUMN partner_id;

DROP INDEX IF EXISTS idx_oauths_user_id;
DROP INDEX IF EXISTS idx_oauths_provider_account_id;
DROP INDEX IF EXISTS idx_oauths_provider_account_name;
