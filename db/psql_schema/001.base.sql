-- user stuff

CREATE TABLE IF NOT EXISTS users (
	id 					bigserial 	PRIMARY KEY,
	created_at			timestamptz	NOT NULL DEFAULT NOW(),
	name 				text 	  	NOT NULL UNIQUE,
	admin 				boolean 	NOT NULL DEFAULT false,
	proposer 			boolean		NOT NULL DEFAULT false,
	email 				text 	  	NOT NULL UNIQUE,
	password 			text 	  	NOT NULL,
	bio 				text 		NOT NULL DEFAULT '',
    generated           boolean     NOT NULL DEFAULT false,
	
	verified_email 		boolean		NOT NULL DEFAULT false,
	email_verif_sent_at timestamptz,

	preferred_language text NOT NULL DEFAULT 'ro'
);

-- problem stuff

CREATE TABLE IF NOT EXISTS problems (
	id 			  	bigserial 			PRIMARY KEY,
	created_at 	  	timestamptz 		NOT NULL DEFAULT NOW(),
	name 		  	text 	    		NOT NULL,
	description   	text				NOT NULL DEFAULT '',
	test_name 	  	text      			NOT NULL,
	time_limit 	  	double precision 	NOT NULL DEFAULT 0.1,
	memory_limit  	integer   			NOT NULL DEFAULT 65536,

	source_size   	integer   			NOT NULL DEFAULT 10000,
	console_input 	boolean 			NOT NULL DEFAULT false,
	visible 	  	boolean 			NOT NULL DEFAULT false,

	source_credits 	text				NOT NULL DEFAULT '',
	author_credits 	text				NOT NULL DEFAULT '',
	short_description text				NOT NULL DEFAULT '',
	default_points 	integer 			NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS tests (
	id 			bigserial  		PRIMARY KEY,
	created_at 	timestamptz		NOT NULL DEFAULT NOW(),
	score 		integer 		NOT NULL,
	problem_id  bigint			NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible_id  bigint 			NOT NULL,
	orphaned 	bool 			NOT NULL DEFAULT false
);

-- submissions stuff

CREATE TYPE status AS ENUM (
	'creating',
	'waiting',
	'working',
	'finished'
);

CREATE TABLE IF NOT EXISTS submissions (
	id 				bigserial 		PRIMARY KEY,
	created_at 		timestamptz		NOT NULL DEFAULT NOW(),
	user_id 		bigint			NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	problem_id 		bigint  		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    contest_id      bigint          REFERENCES contests(id) ON DELETE SET NULL,
	language		text 			NOT NULL,
	code 			text 			NOT NULL,
	status 			status 			NOT NULL DEFAULT 'creating',
	compile_error 	boolean,
	compile_message text,
	score 			integer			NOT NULL DEFAULT 0,
	max_time 		DOUBLE PRECISION NOT NULL DEFAULT -1,
	max_memory 		INTEGER 		NOT NULL DEFAULT -1,
    code_size       INTEGER         NOT NULL GENERATED ALWAYS AS (length(code)) STORED
);

CREATE TABLE IF NOT EXISTS submission_tests (
	id 				bigserial			PRIMARY KEY,
	created_at 		timestamptz			NOT NULL DEFAULT NOW(),
	done			boolean 			NOT NULL DEFAULT false,
	verdict 		text         		NOT NULL DEFAULT '',
	time 			double precision	NOT NULL DEFAULT 0,
	memory 			integer				NOT NULL DEFAULT 0,
	score 			integer				NOT NULL DEFAULT 0,
	test_id			bigint  			NOT NULL REFERENCES tests(id) ON DELETE CASCADE,
	user_id 		bigint  			NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	submission_id 	bigint  			NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,

    -- copied from problem test
    visible_id      bigint              NOT NULL,
    max_score       bigint              NOT NULL
);


-- TODO: Now that we have all the necessary attributes for a subtest copied into the subtest
-- Can't we update test_id to be nullable with on delete set default and remove all the orphaned cruft?


CREATE TABLE IF NOT EXISTS submission_subtasks (
    id              bigserial       PRIMARY KEY,
    created_at      timestamptz     NOT NULL DEFAULT NOW(),
    submission_id   bigint          NOT NULL REFERENCES submissions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    
    -- user_id might be useful in the future
    -- if I ever implement a cross-submission total score system
    user_id         bigint          NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    
    -- Might be useful to also store subtask id, just in case, even if it may never be properly used
    subtask_id      bigint          REFERENCES subtasks(id) ON DELETE SET NULL ON UPDATE CASCADE,

    -- Copy attributes from subtask
	problem_id 	    bigint 		    NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible_id 	    integer 	    NOT NULL,
	score 		    integer 	    NOT NULL,
    UNIQUE (submission_id, subtask_id)
);

CREATE TABLE IF NOT EXISTS submission_subtask_subtests (
    submission_subtask_id   bigint NOT NULL REFERENCES submission_subtasks(id) ON DELETE CASCADE ON UPDATE CASCADE,
    submission_test_id      bigint NOT NULL REFERENCES submission_tests(id) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE (submission_subtask_id, submission_test_id)
);

CREATE TABLE IF NOT EXISTS submission_pastes (
    paste_id        text    NOT NULL UNIQUE,
    submission_id   bigint  NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    author_id       bigint  NOT NULL REFERENCES users(id) ON DELETE CASCADE 
);

CREATE TABLE IF NOT EXISTS problem_lists (
	id 			bigserial 	PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	author_id 	bigint 		NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	title 		text 		NOT NULL DEFAULT '',
	description text 		NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS problem_list_problems (
    pblist_id bigint NOT NULL REFERENCES problem_lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    problem_id bigint NOT NULL REFERENCES problems(id) ON DELETE CASCADE ON UPDATE CASCADE,
    position bigint NOT NULL DEFAULT 0,
    UNIQUE (pblist_id, problem_id)
);

CREATE TABLE IF NOT EXISTS problem_list_pblists (
    parent_id bigint NOT NULL REFERENCES problem_lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    child_id bigint NOT NULL REFERENCES problem_lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    position bigint NOT NULL DEFAULT 0,
    UNIQUE (parent_id, child_id),
	CHECK (parent_id != child_id)
);



CREATE TABLE IF NOT EXISTS subtasks (
	id 			bigserial  	PRIMARY KEY,
	created_at  timestamptz NOT NULL DEFAULT NOW(),
	problem_id 	bigint 		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible_id 	integer 	NOT NULL,
	score 		integer 	NOT NULL
);

CREATE TABLE IF NOT EXISTS subtask_tests (
    subtask_id bigint NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE ON UPDATE CASCADE,
    test_id bigint NOT NULL REFERENCES tests(id) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE (subtask_id, test_id)
);

CREATE TABLE IF NOT EXISTS sessions (
	id 			text 		PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	user_id 	integer 	NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS verifications (
	id 			text 		PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	user_id 	integer 	NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS pwd_reset_requests (
	id 			text 		PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	user_id 	integer 	NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS attachments (
	id 			bigserial 	PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	problem_id 	bigint 		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible 	boolean 	NOT NULL DEFAULT true,
	private 	boolean 	NOT NULL DEFAULT false,

	name 		text 		NOT NULL,
	data 		bytea 		NOT NULL,
	data_size 	INTEGER 	GENERATED ALWAYS AS (length(data)) STORED
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id          bigserial 	    PRIMARY KEY,
    logged_at   timestamptz     NOT NULL DEFAULT NOW(),
    system_log  boolean         NOT NULL DEFAULT false,
    msg         text            NOT NULL DEFAULT '',
    author_id   bigint          DEFAULT null REFERENCES users(id) ON DELETE SET NULL
);


CREATE TYPE pbaccess_type AS ENUM (
    'editor',
    'viewer'
); 

CREATE TABLE IF NOT EXISTS problem_user_access (
    problem_id  bigint           NOT NULL REFERENCES problems(id) ON DELETE CASCADE ON UPDATE CASCADE,
    user_id     bigint           NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    access      pbaccess_type    NOT NULL,

    UNIQUE (problem_id, user_id)
);

-- contests

CREATE TABLE IF NOT EXISTS contests (
	id 					bigserial 	PRIMARY KEY,
	created_at			timestamptz	NOT NULL DEFAULT NOW(),
	name 				text 	  	NOT NULL UNIQUE,

    public_join         boolean     NOT NULL DEFAULT true,
    start_time          timestamptz NOT NULL,
    end_time            timestamptz NOT NULL,
    max_sub_count       integer     NOT NULL DEFAULT 30,

    virtual             boolean     NOT NULL DEFAULT false,
    visible             boolean     NOT NULL DEFAULT false,

    CHECK (start_time <= end_time)
);

CREATE TABLE IF NOT EXISTS contest_user_access (
    contest_id  bigint           NOT NULL REFERENCES contests(id) ON DELETE CASCADE ON UPDATE CASCADE,
    user_id     bigint           NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    access      pbaccess_type    NOT NULL,

    UNIQUE (contest_id, user_id)
);

CREATE TABLE IF NOT EXISTS contest_problems (
    contest_id bigint NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    problem_id bigint NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    position bigint NOT NULL DEFAULT 0,
    UNIQUE (contest_id, problem_id)
);

CREATE TABLE IF NOT EXISTS contest_registrations (
	created_at	    timestamptz  NOT NULL DEFAULT NOW(),
    user_id         bigint       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contest_id      bigint       NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    UNIQUE (user_id, contest_id)
);

CREATE TABLE IF NOT EXISTS contest_questions (
    id             bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    author_id      bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contest_id     bigint NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    question       text   NOT NULL,
    created_at 	   timestamptz NOT NULL DEFAULT NOW(),

    responded_at   timestamptz,
    response       text
);

CREATE TABLE IF NOT EXISTS contest_announcements (
    id             bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    contest_id     bigint NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    announcement   text   NOT NULL,
    created_at 	   timestamptz NOT NULL DEFAULT NOW()
);


-- views

CREATE OR REPLACE VIEW max_score_view (user_id, problem_id, score)
    AS SELECT user_id, problem_id, MAX(score) AS score FROM submissions
    GROUP BY problem_id, user_id ORDER BY problem_id, user_id;

CREATE OR REPLACE VIEW problem_viewers AS
    (SELECT pbs.id as problem_id, 0 as user_id
        FROM problems pbs, users
        WHERE pbs.visible = true) -- Base case, problem is visible
    UNION ALL
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM problems pbs, users 
        WHERE pbs.visible = true OR users.admin = true) -- Problem is visible or user is admin
    UNION ALL
    (SELECT problem_id, user_id FROM problem_user_access) -- Problem editors/viewers
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id 
        FROM contest_problems pbs, contest_user_access users 
        WHERE pbs.contest_id = users.contest_id) -- Contest editors/viewers
    UNION ALL
    (SELECT pbs.problem_id as problem_id, 0 as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.start_time <= NOW() AND NOW() <= contests.end_time) -- Visible running contests for anons. TODO: Find alternative
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.id as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.start_time <= NOW() AND NOW() <= contests.end_time) -- Visible running contests
    UNION ALL
    (SELECT pbs.problem_id as problem_id, 0 as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.end_time <= NOW()) -- Visible contests after they ended for anons. TODO: Find alternative
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.id as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.end_time <= NOW()) -- Visible contests after they ended
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id
        FROM contest_problems pbs, contest_registrations users, contests 
        WHERE pbs.contest_id = contests.id AND contests.id = users.contest_id
        AND contests.visible = false 
        AND contests.start_time <= NOW() AND NOW() <= contests.end_time); -- Contest registrants during the contest for hidden running contest

CREATE OR REPLACE VIEW problem_editors AS
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM problems pbs, users 
        WHERE users.admin = true) -- User is admin
    UNION ALL
    (SELECT problem_id, user_id FROM problem_user_access WHERE access = 'editor') -- Problem editors
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id 
        FROM contest_problems pbs, contest_user_access users 
        WHERE pbs.contest_id = users.contest_id AND users.access = 'editor'); -- Contest editors

-- Cases where a contest should be visible:
--   - It's visible
--   - Admins
--   - Testers/Editors
--   - It's not visible but it's running and user is registered

CREATE OR REPLACE VIEW running_contests AS (
    SELECT * from contests WHERE contests.start_time <= NOW() AND NOW() <= contests.end_time
);

CREATE OR REPLACE VIEW contest_visibility AS (
    (SELECT contests.id AS contest_id, 0 AS user_id FROM contests 
        WHERE contests.visible = true) -- visible to anonymous users
    UNION
    (SELECT contests.id AS contest_id, users.id AS user_id FROM contests, users 
        WHERE contests.visible = true) -- visible to logged in users
    UNION
    (SELECT contests.id AS contest_id, users.id AS user_id FROM contests, users 
        WHERE users.admin = true) -- admin
    UNION
    (SELECT contest_id, user_id FROM contest_user_access) -- Testers/Editors
    UNION
    (SELECT contests.id AS contest_id, users.user_id AS user_id FROM contests, contest_registrations users 
        WHERE contests.id = users.contest_id AND contests.visible = false) -- not visible but registered
);

CREATE OR REPLACE VIEW max_score_contest_view
    AS SELECT subs.user_id, subs.problem_id, subs.contest_id, MAX(subs.score) AS score FROM submissions subs, contest_registrations users, contest_problems pbs 
    WHERE subs.contest_id IS NOT NULL -- actually from contest
    AND users.user_id = subs.user_id AND users.contest_id = subs.contest_id -- Registered users
    AND pbs.problem_id = subs.problem_id AND pbs.contest_id = subs.contest_id -- Existent problems
    GROUP BY subs.problem_id, subs.contest_id, subs.user_id ORDER BY subs.contest_id, subs.user_id, subs.problem_id;

CREATE OR REPLACE VIEW contest_top_view
    AS WITH contest_scores AS (
        SELECT user_id, contest_id, SUM(score) AS total_score FROM max_score_contest_view GROUP BY user_id, contest_id
    ) SELECT users.user_id, users.contest_id, COALESCE(scores.total_score, 0) AS total_score 
    FROM contest_registrations users LEFT OUTER JOIN contest_scores scores ON users.user_id = scores.user_id AND users.contest_id = scores.contest_id ORDER BY contest_id, total_score DESC, user_id;


-- indexes


---- pblists

CREATE INDEX IF NOT EXISTS pblist_problems_index ON problem_list_problems (pblist_id);
CREATE INDEX IF NOT EXISTS pblist_pblists_index ON problem_list_pblists (parent_id);

---- submissions

CREATE INDEX IF NOT EXISTS problem_user_submissions_index ON submissions (user_id, problem_id);
CREATE INDEX IF NOT EXISTS problem_submissions_index ON submissions (problem_id);
CREATE INDEX IF NOT EXISTS contest_submissions_index ON submissions (contest_id);

CREATE INDEX IF NOT EXISTS submission_subtests_index ON submission_tests (submission_id);
CREATE INDEX IF NOT EXISTS submission_subtasks_index ON submission_subtasks (submission_id);

---- problems

CREATE INDEX IF NOT EXISTS problem_access_index ON problem_user_access (problem_id);
CREATE INDEX IF NOT EXISTS problem_attachments_index ON attachments (problem_id);
CREATE INDEX IF NOT EXISTS problem_tests_index ON tests (problem_id);

---- contests

CREATE INDEX IF NOT EXISTS contest_access_index ON contest_user_access (contest_id);
CREATE INDEX IF NOT EXISTS contest_problems_index ON contest_problems (contest_id);
CREATE INDEX IF NOT EXISTS contest_registrations_index ON contest_registrations (contest_id);
CREATE INDEX IF NOT EXISTS contest_questions_index ON contest_questions (contest_id);
CREATE INDEX IF NOT EXISTS contest_announcements_index ON contest_announcements (contest_id);

