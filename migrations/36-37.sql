CREATE TABLE cached_data (
	"key" int2 NOT NULL,
	event_id int4 NOT NULL,
	"data" bytea NOT NULL,
	"timestamp" timestamptz NOT NULL,
	CONSTRAINT cached_data_pk PRIMARY KEY ("key",event_id),
	CONSTRAINT cached_data_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);
