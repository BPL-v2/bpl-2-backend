CREATE SCHEMA IF NOT EXISTS bpl2;
CREATE TABLE migrations (
    id INT NOT NULL,
    "timestamp" timestamptz NOT NULL,
    CONSTRAINT migrations_pkey PRIMARY KEY (id)
);