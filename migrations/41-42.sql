CREATE TABLE atlas_trees (
	event_id int4 NOT NULL,
	user_id int4 NOT NULL,
	"index" int2 NOT NULL,
	nodes _int2 NOT NULL,
	"timestamp" timestamptz NOT NULL DEFAULT current_timestamp,
	CONSTRAINT atlas_trees_users_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT atlas_trees_events_fk FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX atlas_trees_event_id_idx ON atlas_trees (event_id);
CREATE INDEX atlas_trees_user_id_idx ON atlas_trees (user_id);

INSERT INTO atlas_trees (event_id, user_id, "index", nodes, "timestamp")
SELECT 
    a.event_id,
    a.user_id,
    0 as "index",
    ARRAY(SELECT (unnest(a.tree1) - 32768)::int2) as nodes,
    e.event_end_time as "timestamp"
FROM atlas a
JOIN events e ON a.event_id = e.id
WHERE array_length(a.tree1, 1) > 0;

INSERT INTO atlas_trees (event_id, user_id, "index", nodes, "timestamp")
SELECT 
    a.event_id,
    a.user_id,
    1 as "index",
    ARRAY(SELECT (unnest(a.tree2) - 32768)::int2) as nodes,
    e.event_end_time as "timestamp"
FROM atlas a
JOIN events e ON a.event_id = e.id
WHERE array_length(a.tree2, 1) > 0;

INSERT INTO atlas_trees (event_id, user_id, "index", nodes, "timestamp")
SELECT 
    a.event_id,
    a.user_id,
    2 as "index",
    ARRAY(SELECT (unnest(a.tree3) - 32768)::int2) as nodes,
    e.event_end_time as "timestamp"
FROM atlas a
JOIN events e ON a.event_id = e.id
WHERE array_length(a.tree3, 1) > 0;

DROP TABLE atlas;