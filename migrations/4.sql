-- Add the partner_wish column
ALTER TABLE bpl2.signups ADD COLUMN partner_wish TEXT;

-- Populate partner_wish with the PoE name from the partner's oauth account
UPDATE bpl2.signups s
SET partner_wish = o.name
FROM bpl2.oauths o
WHERE o.user_id = s.partner_id
  AND o.provider = 'poe'
  AND s.partner_id IS NOT NULL;

ALTER TABLE bpl2.signups DROP COLUMN partner_id;