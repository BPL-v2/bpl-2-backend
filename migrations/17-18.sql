CREATE TABLE change_ids (
    current_change_id text NOT NULL,
    next_change_id text NOT NULL,
    event_id bigserial NOT NULL,
    CONSTRAINT change_ids_pkey PRIMARY KEY (event_id)
);
INSERT INTO change_ids (current_change_id, next_change_id, event_id)
SELECT -- The second to last change_id (rn = 2) as current
    MAX(
        CASE
            WHEN rn = 2 THEN next_change_id
        END
    ) AS current_change_id,
    -- The last change_id (rn = 1) as next
    MAX(
        CASE
            WHEN rn = 1 THEN next_change_id
        END
    ) AS next_change_id,
    event_id
FROM (
        SELECT next_change_id,
            event_id,
            ROW_NUMBER() OVER (
                PARTITION BY event_id
                ORDER BY "timestamp" DESC
            ) AS rn
        FROM stash_changes
    ) ranked
WHERE rn <= 2
GROUP BY event_id;
ALTER TABLE stash_changes DROP COLUMN next_change_id;