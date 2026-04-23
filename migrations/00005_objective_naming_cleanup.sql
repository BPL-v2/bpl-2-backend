-- +goose Up

-- Removed concepts per proposal.
DELETE FROM scoring_presets WHERE scoring_method = 'BINGO_3';
DELETE FROM objectives WHERE aggregation = 'SUM_LATEST';

UPDATE objectives
SET aggregation = CASE aggregation
	WHEN 'LATEST' THEN 'LATEST_VALUE'
	WHEN 'EARLIEST' THEN 'FIRST_COMPLETION'
	WHEN 'EARLIEST_FRESH_ITEM' THEN 'FIRST_FRESH_COMPLETION'
	WHEN 'MAXIMUM' THEN 'HIGHEST_VALUE'
	WHEN 'MINIMUM' THEN 'LOWEST_VALUE'
	WHEN 'DIFFERENCE_BETWEEN' THEN 'VALUE_CHANGE_IN_WINDOW'
	WHEN 'NONE' THEN 'CHILD_RESULT'
	ELSE aggregation
END;

UPDATE objectives
SET number_field = CASE number_field
	WHEN 'PLAYER_LEVEL' THEN 'CHARACTER_LEVEL'
	WHEN 'DELVE_DEPTH_PAST_100' THEN 'DELVE_DEPTH_AFTER_100'
	WHEN 'PROGRESSIVE_DELVE_DEPTH' THEN 'WEIGHTED_DELVE_DEPTH'
	WHEN 'PANTHEON' THEN 'TEAM_PLAYERS_WITH_PANTHEON_UNLOCKED'
	WHEN 'ASCENDANCY' THEN 'ASCENDANCY_POINTS'
	WHEN 'FULLY_ASCENDED' THEN 'TEAM_PLAYERS_WITH_ALL_LABS_COMPLETED'
	WHEN 'BLOODLINE_ASCENDANCY' THEN 'BLOODLINE_ASCENDANCY_UNLOCKED'
	WHEN 'PLAYER_SCORE' THEN 'PERSONAL_OBJECTIVE_SCORE'
	WHEN 'VOID_STONES' THEN 'VOID_STONE_COUNT'
	WHEN 'INC_MOVEMENT_SPEED' THEN 'MOVEMENT_SPEED_BONUS'
	WHEN 'PHYS_MAX_HIT' THEN 'PHYSICAL_MAX_HIT'
	WHEN 'ELE_MAX_HIT' THEN 'ELEMENTAL_MAX_HIT'
	WHEN 'ATTACK_BLOCK' THEN 'ATTACK_BLOCK_CHANCE'
	WHEN 'SPELL_BLOCK' THEN 'SPELL_BLOCK_CHANCE'
	WHEN 'HIGH_ILVL_FLASKS' THEN 'HIGH_ITEM_LEVEL_FLASK_COUNT'
	WHEN 'ELE_MAX_RES' THEN 'LOWEST_ELEMENTAL_RESISTANCE'
	WHEN 'INFLUENCE_EQUIPPED' THEN 'INFLUENCED_ITEM_COUNT'
	WHEN 'FOULBORN_EQUIPPED' THEN 'FOULBORN_ITEM_COUNT'
	WHEN 'GEMS_EQUIPPED' THEN 'SOCKETED_GEM_COUNT'
	WHEN 'CORRUPTED_ITEMS_EQUIPPED' THEN 'CORRUPTED_ITEM_COUNT'
	WHEN 'JEWELS_WITH_IMPLICITS_EQUIPPED' THEN 'JEWELS_WITH_IMPLICITS_COUNT'
	WHEN 'ENCHANTED_ITEMS_EQUIPPED' THEN 'ENCHANTED_ITEM_COUNT'
	WHEN 'SUBMISSION_VALUE' THEN 'SUBMITTED_VALUE'
	WHEN 'FINISHED_OBJECTIVES' THEN 'COMPLETED_CHILD_OBJECTIVE_COUNT'
	ELSE number_field
END;

UPDATE scoring_presets
SET scoring_method = CASE scoring_method
	WHEN 'PRESENCE' THEN 'FIXED_POINTS_ON_COMPLETION'
	WHEN 'POINTS_FROM_VALUE' THEN 'POINTS_BY_VALUE'
	WHEN 'RANKED_TIME' THEN 'RANK_BY_COMPLETION_TIME'
	WHEN 'RANKED_VALUE' THEN 'RANK_BY_HIGHEST_VALUE'
	WHEN 'RANKED_REVERSE' THEN 'RANK_BY_LOWEST_VALUE'
	WHEN 'RANKED_COMPLETION_TIME' THEN 'RANK_BY_CHILD_COMPLETION_TIME'
	WHEN 'BONUS_PER_COMPLETION' THEN 'BONUS_PER_CHILD_COMPLETION'
	WHEN 'BINGO_BOARD' THEN 'BINGO_BOARD_RANKING'
	WHEN 'CHILD_NUMBER_SUM' THEN 'RANK_BY_CHILD_VALUE_SUM'
	ELSE scoring_method
END;

UPDATE scoring_presets
SET extra = COALESCE(extra, '{}'::jsonb);

UPDATE scoring_presets
SET extra = (extra - 'required_child_completions') || jsonb_build_object('required_completed_children', extra->'required_child_completions')
WHERE extra ? 'required_child_completions';

UPDATE scoring_presets
SET extra = (extra - 'required_child_completions_percent') || jsonb_build_object('required_completed_children_percent', extra->'required_child_completions_percent')
WHERE extra ? 'required_child_completions_percent';

UPDATE scoring_presets
SET extra = (extra - 'required_number_of_bingos') || jsonb_build_object('required_bingo_count', extra->'required_number_of_bingos')
WHERE extra ? 'required_number_of_bingos';

ALTER TABLE objectives RENAME COLUMN number_field TO tracked_value;
ALTER TABLE objectives RENAME COLUMN aggregation TO counting_method;
ALTER TABLE objectives RENAME COLUMN number_field_explanation TO tracked_value_explanation;

ALTER TABLE scoring_presets RENAME COLUMN scoring_method TO scoring_rule;
ALTER TABLE scoring_presets RENAME TO scoring_rules;
ALTER INDEX scoring_presets_event_id_idx RENAME TO scoring_rules_event_id_idx;
ALTER TABLE scoring_rules RENAME CONSTRAINT scoring_presets_pkey TO scoring_rules_pkey;
ALTER TABLE scoring_rules RENAME CONSTRAINT scoring_presets_event_id_fkey TO scoring_rules_event_id_fkey;

ALTER TABLE objective_scoring_presets RENAME TO objective_scoring_rules;
ALTER TABLE objective_scoring_rules RENAME COLUMN scoring_preset_id TO scoring_rule_id;
ALTER INDEX objective_scoring_presets_scoring_preset_id_idx RENAME TO objective_scoring_rules_scoring_rule_id_idx;
ALTER TABLE objective_scoring_rules RENAME CONSTRAINT objective_scoring_presets_pkey TO objective_scoring_rules_pkey;
ALTER TABLE objective_scoring_rules RENAME CONSTRAINT fk_objective_scoring_presets_objective TO fk_objective_scoring_rules_objective;
ALTER TABLE objective_scoring_rules RENAME CONSTRAINT fk_objective_scoring_presets_scoring_preset TO fk_objective_scoring_rules_scoring_rule;

-- +goose Down

ALTER TABLE objective_scoring_rules RENAME CONSTRAINT fk_objective_scoring_rules_scoring_rule TO fk_objective_scoring_presets_scoring_preset;
ALTER TABLE objective_scoring_rules RENAME CONSTRAINT fk_objective_scoring_rules_objective TO fk_objective_scoring_presets_objective;
ALTER TABLE objective_scoring_rules RENAME CONSTRAINT objective_scoring_rules_pkey TO objective_scoring_presets_pkey;
ALTER INDEX objective_scoring_rules_scoring_rule_id_idx RENAME TO objective_scoring_presets_scoring_preset_id_idx;
ALTER TABLE objective_scoring_rules RENAME COLUMN scoring_rule_id TO scoring_preset_id;
ALTER TABLE objective_scoring_rules RENAME TO objective_scoring_presets;

ALTER TABLE scoring_rules RENAME CONSTRAINT scoring_rules_event_id_fkey TO scoring_presets_event_id_fkey;
ALTER TABLE scoring_rules RENAME CONSTRAINT scoring_rules_pkey TO scoring_presets_pkey;
ALTER INDEX scoring_rules_event_id_idx RENAME TO scoring_presets_event_id_idx;
ALTER TABLE scoring_rules RENAME TO scoring_presets;
ALTER TABLE scoring_presets RENAME COLUMN scoring_rule TO scoring_method;

ALTER TABLE objectives RENAME COLUMN tracked_value_explanation TO number_field_explanation;
ALTER TABLE objectives RENAME COLUMN counting_method TO aggregation;
ALTER TABLE objectives RENAME COLUMN tracked_value TO number_field;

UPDATE scoring_presets
SET extra = (extra - 'required_bingo_count') || jsonb_build_object('required_number_of_bingos', extra->'required_bingo_count')
WHERE extra ? 'required_bingo_count';

UPDATE scoring_presets
SET extra = (extra - 'required_completed_children_percent') || jsonb_build_object('required_child_completions_percent', extra->'required_completed_children_percent')
WHERE extra ? 'required_completed_children_percent';

UPDATE scoring_presets
SET extra = (extra - 'required_completed_children') || jsonb_build_object('required_child_completions', extra->'required_completed_children')
WHERE extra ? 'required_completed_children';

UPDATE scoring_presets
SET scoring_method = CASE scoring_method
	WHEN 'FIXED_POINTS_ON_COMPLETION' THEN 'PRESENCE'
	WHEN 'POINTS_BY_VALUE' THEN 'POINTS_FROM_VALUE'
	WHEN 'RANK_BY_COMPLETION_TIME' THEN 'RANKED_TIME'
	WHEN 'RANK_BY_HIGHEST_VALUE' THEN 'RANKED_VALUE'
	WHEN 'RANK_BY_LOWEST_VALUE' THEN 'RANKED_REVERSE'
	WHEN 'RANK_BY_CHILD_COMPLETION_TIME' THEN 'RANKED_COMPLETION_TIME'
	WHEN 'BONUS_PER_CHILD_COMPLETION' THEN 'BONUS_PER_COMPLETION'
	WHEN 'BINGO_BOARD_RANKING' THEN 'BINGO_BOARD'
	WHEN 'RANK_BY_CHILD_VALUE_SUM' THEN 'CHILD_NUMBER_SUM'
	ELSE scoring_method
END;

UPDATE objectives
SET number_field = CASE number_field
	WHEN 'CHARACTER_LEVEL' THEN 'PLAYER_LEVEL'
	WHEN 'DELVE_DEPTH_AFTER_100' THEN 'DELVE_DEPTH_PAST_100'
	WHEN 'WEIGHTED_DELVE_DEPTH' THEN 'PROGRESSIVE_DELVE_DEPTH'
	WHEN 'TEAM_PLAYERS_WITH_PANTHEON_UNLOCKED' THEN 'PANTHEON'
	WHEN 'ASCENDANCY_POINTS' THEN 'ASCENDANCY'
	WHEN 'TEAM_PLAYERS_WITH_ALL_LABS_COMPLETED' THEN 'FULLY_ASCENDED'
	WHEN 'BLOODLINE_ASCENDANCY_UNLOCKED' THEN 'BLOODLINE_ASCENDANCY'
	WHEN 'PERSONAL_OBJECTIVE_SCORE' THEN 'PLAYER_SCORE'
	WHEN 'VOID_STONE_COUNT' THEN 'VOID_STONES'
	WHEN 'MOVEMENT_SPEED_BONUS' THEN 'INC_MOVEMENT_SPEED'
	WHEN 'PHYSICAL_MAX_HIT' THEN 'PHYS_MAX_HIT'
	WHEN 'ELEMENTAL_MAX_HIT' THEN 'ELE_MAX_HIT'
	WHEN 'ATTACK_BLOCK_CHANCE' THEN 'ATTACK_BLOCK'
	WHEN 'SPELL_BLOCK_CHANCE' THEN 'SPELL_BLOCK'
	WHEN 'HIGH_ITEM_LEVEL_FLASK_COUNT' THEN 'HIGH_ILVL_FLASKS'
	WHEN 'LOWEST_ELEMENTAL_RESISTANCE' THEN 'ELE_MAX_RES'
	WHEN 'INFLUENCED_ITEM_COUNT' THEN 'INFLUENCE_EQUIPPED'
	WHEN 'FOULBORN_ITEM_COUNT' THEN 'FOULBORN_EQUIPPED'
	WHEN 'SOCKETED_GEM_COUNT' THEN 'GEMS_EQUIPPED'
	WHEN 'CORRUPTED_ITEM_COUNT' THEN 'CORRUPTED_ITEMS_EQUIPPED'
	WHEN 'JEWELS_WITH_IMPLICITS_COUNT' THEN 'JEWELS_WITH_IMPLICITS_EQUIPPED'
	WHEN 'ENCHANTED_ITEM_COUNT' THEN 'ENCHANTED_ITEMS_EQUIPPED'
	WHEN 'SUBMITTED_VALUE' THEN 'SUBMISSION_VALUE'
	WHEN 'COMPLETED_CHILD_OBJECTIVE_COUNT' THEN 'FINISHED_OBJECTIVES'
	ELSE number_field
END;

UPDATE objectives
SET aggregation = CASE aggregation
	WHEN 'LATEST_VALUE' THEN 'LATEST'
	WHEN 'FIRST_COMPLETION' THEN 'EARLIEST'
	WHEN 'FIRST_FRESH_COMPLETION' THEN 'EARLIEST_FRESH_ITEM'
	WHEN 'HIGHEST_VALUE' THEN 'MAXIMUM'
	WHEN 'LOWEST_VALUE' THEN 'MINIMUM'
	WHEN 'VALUE_CHANGE_IN_WINDOW' THEN 'DIFFERENCE_BETWEEN'
	WHEN 'CHILD_RESULT' THEN 'NONE'
	ELSE aggregation
END;

-- Deleted rows for SUM_LATEST and BINGO_3 are intentionally not restored.
