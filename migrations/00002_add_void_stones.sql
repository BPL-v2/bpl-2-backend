-- +goose Up
ALTER TABLE characters ADD COLUMN void_stones text[] NOT NULL DEFAULT '{}';
ALTER TABLE character_pobs ADD COLUMN attack_block int2 NOT NULL DEFAULT 0;
ALTER TABLE character_pobs ADD COLUMN spell_block int2 NOT NULL DEFAULT 0;
ALTER TABLE character_pobs ADD COLUMN lowest_ele_res int2 NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE characters DROP COLUMN void_stones;
ALTER TABLE character_pobs DROP COLUMN attack_block;
ALTER TABLE character_pobs DROP COLUMN spell_block;
ALTER TABLE character_pobs DROP COLUMN lowest_ele_res;
