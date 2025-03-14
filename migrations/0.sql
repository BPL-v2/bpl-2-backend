CREATE SCHEMA public if not exists;
CREATE TABLE public.migrations (
    id INT NOT NULL,
    "timestamp" timestamptz NOT NULL,
    CONSTRAINT migrations_pkey PRIMARY KEY (id)
);