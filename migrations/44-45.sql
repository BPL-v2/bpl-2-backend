-- Create the new table with bytea instead of text for export
CREATE TABLE bpl2.character_pobs_binary (
    id bigserial NOT NULL,
    character_id text NOT NULL,
    export bytea NOT NULL,
    "level" int2 NOT NULL,
    ascendancy text NOT NULL,
    main_skill text NOT NULL,
    "timestamp" timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT character_pobs_binary_pkey PRIMARY KEY (id)
);
CREATE INDEX character_pobs_binary_character_id_idx ON bpl2.character_pobs_binary USING btree (character_id);

-- Add foreign key
ALTER TABLE bpl2.character_pobs_binary ADD CONSTRAINT character_pobs_binary_character_id_fkey 
FOREIGN KEY (character_id) REFERENCES bpl2."characters"(id) ON DELETE CASCADE ON UPDATE CASCADE;

-- Create function to convert the custom base64 format to bytea
CREATE OR REPLACE FUNCTION convert_pob_export_to_bytea(export_text text)
RETURNS bytea AS $$
DECLARE
    normalized_base64 text;
BEGIN
    -- Replace - with + and _ with / (reverse of the encoding)
    normalized_base64 := replace(replace(export_text, '-', '+'), '_', '/');
    
    -- Decode from base64 to bytea
    RETURN decode(normalized_base64, 'base64');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Migrate data from old table to new table
INSERT INTO bpl2.character_pobs_binary (id, character_id, export, "level", ascendancy, main_skill, "timestamp")
SELECT 
    id,
    character_id,
    convert_pob_export_to_bytea(export) as export,
    "level",
    ascendancy,
    main_skill,
    "timestamp"
FROM bpl2.character_pobs;

-- Update the sequence to continue from the max id
SELECT setval('bpl2.character_pobs_binary_id_seq', (SELECT COALESCE(MAX(id), 1) FROM bpl2.character_pobs_binary));

-- Drop the old table and rename the new one
DROP TABLE bpl2.character_pobs CASCADE;
ALTER TABLE bpl2.character_pobs_binary RENAME TO character_pobs;
ALTER TABLE bpl2.character_pobs RENAME CONSTRAINT character_pobs_binary_pkey TO character_pobs_pkey;
ALTER INDEX character_pobs_binary_character_id_idx RENAME TO character_pobs_character_id_idx;
ALTER TABLE bpl2.character_pobs RENAME CONSTRAINT character_pobs_binary_character_id_fkey TO character_pobs_character_id_fkey;
ALTER SEQUENCE bpl2.character_pobs_binary_id_seq RENAME TO character_pobs_id_seq;