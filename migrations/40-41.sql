-- Remove user_ids from objective matches from guild stashes
UPDATE objective_matches 
SET user_id = NULL 
WHERE stash_change_id IN (
    SELECT id 
    FROM stash_changes 
    WHERE LENGTH(stash_id) = 10
);