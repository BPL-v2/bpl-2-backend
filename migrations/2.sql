ALTER TABLE objectives ADD COLUMN conditions jsonb;

UPDATE objectives o
SET conditions = (
    SELECT jsonb_agg(
        jsonb_build_object(
            'field', c.field,
            'operator', c.operator,
            'value', c.value
        )
    )
    FROM conditions c
    WHERE c.objective_id = o.id
);

DROP TABLE conditions;