package postgres

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(databaseUrl string) *Repository {
	if databaseUrl == "" {
		panic("Database URL is empty")
	}

	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		panic("Failed to open database: " + err.Error())
	}

	if err := db.Ping(); err != nil {
		panic("Failed to connect to the database: " + err.Error())
	}

	runTableMigrations(db)

	return &Repository{db: db}
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func rollbackTx(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		log.Printf("failed to rollback transaction: %v", err)
	}
}

func runTableMigrations(db *sql.DB) {
	// TODO make email unique
	createTableQuery := `

-- USERS TABLE
CREATE TABLE IF NOT EXISTS  users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    fcm_token TEXT DEFAULT NULL,
    password TEXT, -- encrypted
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS oauth (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE NO ACTION,
    oauth_provider TEXT NOT NULL,
    oauth_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (oauth_provider, oauth_id, user_id)
);

-- FILES TABLE
CREATE TABLE IF NOT EXISTS  files (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE NO ACTION,
    title TEXT NOT NULL,
    size BIGINT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL
);

-- REFRESH TOKENS
CREATE TABLE IF NOT EXISTS  refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE NO ACTION,
    token TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    unique (user_id)
);

-- CHATS
CREATE TABLE IF NOT EXISTS  chats (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE NO ACTION,
    title TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- MESSAGES
CREATE TABLE IF NOT EXISTS  messages (
    id UUID PRIMARY KEY,
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE ON UPDATE NO ACTION,
    is_user BOOLEAN NOT NULL,
    content TEXT NOT NULL, 
    created_at TIMESTAMP NOT NULL,
    context_snapshot JSONB,
    attachments JSONB
);

-- QUIZZES
CREATE TABLE IF NOT EXISTS quizzes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    duration_minutes INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

-- QUIZ QUESTIONS
CREATE TABLE IF NOT EXISTS quiz_questions (
    id UUID PRIMARY KEY,
    quiz_id UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    question_no INTEGER NOT NULL,
    question_text TEXT NOT NULL,
    explanation TEXT,  -- Doğru cevap açıklaması
    points INTEGER DEFAULT 1,
    UNIQUE(quiz_id, question_no)
);

-- QUESTION OPTIONS
CREATE TABLE IF NOT EXISTS quiz_question_options (
    id UUID PRIMARY KEY,
    question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
    option_label CHAR(1) NOT NULL,
    option_text TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INTEGER NOT NULL,
    UNIQUE(question_id, option_label),
    CHECK (option_label IN ('A', 'B', 'C', 'D', 'E'))
);

-- QUIZ ATTEMPTS
CREATE TABLE IF NOT EXISTS quiz_attempts (
    id UUID PRIMARY KEY,
    quiz_id UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    score INTEGER,
    total_questions INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS quiz_attempt_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    question_no INT NOT NULL,
    selected_option VARCHAR(1) NOT NULL CHECK (selected_option IN ('A', 'B', 'C', 'D', 'E')),
    correct_option VARCHAR(1) NOT NULL CHECK (correct_option IN ('A', 'B', 'C', 'D', 'E')),
    is_correct BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(attempt_id, question_no)
);

    
    
    -- INDEXES
    CREATE INDEX IF NOT EXISTS  idx_quiz_questions_quiz_id ON quiz_questions(quiz_id);
    CREATE INDEX IF NOT EXISTS  idx_quiz_attempts_quiz_user ON quiz_attempts(quiz_id, user_id);
    CREATE INDEX IF NOT EXISTS idx_quiz_attempt_answers_attempt_id ON quiz_attempt_answers(attempt_id);
	`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		panic("Failed to create tables: " + err.Error())
	}

	// Migrations for existing tables
	migrationQuery := `
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS context_snapshot JSONB;
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachments JSONB;
	ALTER TABLE users ADD COLUMN IF NOT EXISTS preferences JSONB DEFAULT '{"theme":"system","notifications":{"motivation":true,"reminders":true,"updates":true}}';
	`
	_, err = db.Exec(migrationQuery)
	if err != nil {
		panic("Failed to run migrations: " + err.Error())
	}
}
