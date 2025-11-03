CREATE TABLE IF NOT EXISTS objective_validations (
    objective_id int4 NOT NULL,
    item jsonb NOT NULL,
    timestamp timestamptz NOT NULL,
    PRIMARY KEY (objective_id),
    CONSTRAINT objective_validations_objectives_fk FOREIGN KEY (objective_id) REFERENCES bpl2.objectives(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX idx_objective_validations_objective_id ON objective_validations (objective_id);