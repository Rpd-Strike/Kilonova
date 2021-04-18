CREATE TABLE IF NOT EXISTS submission_tests (
	id				INTEGER 	PRIMARY KEY,
	created_at		TIMESTAMP 	NOT NULL DEFAULT CURRENT_TIMESTAMP,
	done			INTEGER 	NOT NULL DEFAULT FALSE,
	verdict			TEXT 		NOT NULL DEFAULT '',
	time			FLOAT 		NOT NULL DEFAULT 0,
	memory			INTEGER		NOT NULL DEFAULT 0,
	score			INTEGER 	NOT NULL DEFAULT 0,
	test_id			INTEGER 	NOT NULL REFERENCES tests(id) ON DELETE CASCADE,
	user_id			INTEGER 	NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	submission_id 	INTEGER 	NOT NULL REFERENCES submissions(id) ON DELETE CASCADE
);
