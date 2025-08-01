ALTER TABLE "characters"
ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE "characters"
ADD old_account_name varchar(255) NULL;
CREATE INDEX characters_old_account_name_idx ON "characters" (old_account_name);