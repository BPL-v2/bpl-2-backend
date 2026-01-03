CREATE TABLE achievements (
    "name" VARCHAR(255) NOT NULL,
    user_id INT NOT NULL REFERENCES users(id),
    PRIMARY KEY (user_id, "name")
);