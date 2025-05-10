DELETE FROM signups
WHERE id NOT IN (
        SELECT MAX(id)
        FROM signups
        GROUP BY user_id,
            event_id
    );
ALTER TABLE signups
ADD CONSTRAINT unique_user_event UNIQUE (user_id, event_id);