ALTER TABLE signups ADD COLUMN partner_id BIGINT;
ALTER TABLE signups ADD CONSTRAINT fk_signup_partner FOREIGN KEY (partner_id) REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX idx_signup_partner ON signups (partner_id);

ALTER TABLE signups DROP CONSTRAINT IF EXISTS unique_user_event;
ALTER TABLE signups DROP COLUMN id;
ALTER TABLE signups ADD CONSTRAINT signups_pkey PRIMARY KEY (event_id, user_id);

ALTER TABLE signups DROP CONSTRAINT IF EXISTS fk_bpl2_signups_user;
ALTER TABLE signups DROP CONSTRAINT IF EXISTS signups_event_id_fkey;
ALTER TABLE signups DROP CONSTRAINT IF EXISTS signups_user_id_fkey;

ALTER TABLE signups ADD CONSTRAINT signups_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE;
ALTER TABLE signups ADD CONSTRAINT signups_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE oauths DROP CONSTRAINT IF EXISTS fk_bpl2_users_oauth_accounts;
ALTER TABLE oauths ADD CONSTRAINT fk_users_oauth_accounts FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX idx_oauths_user_id ON oauths (user_id);
CREATE INDEX idx_oauths_provider_account_id ON oauths (provider, account_id);
CREATE INDEX idx_oauths_provider_account_name ON oauths (provider, name);