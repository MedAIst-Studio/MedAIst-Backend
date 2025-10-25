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
	createTableQuery := `


-- USERS TABLE
CREATE TABLE IF NOT EXISTS  users (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    username TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password TEXT, -- encrypted
    oauth_provider TEXT,
    oauth_id TEXT,
    created_at TIMESTAMP NOT NULL
);

-- FILES TABLE
CREATE TABLE IF NOT EXISTS  files (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    title TEXT NOT NULL,
    size BIGINT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL
);

-- REFRESH TOKENS
CREATE TABLE IF NOT EXISTS  refresh_tokens (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    token TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    unique (user_id)
);

-- CHATS
CREATE TABLE IF NOT EXISTS  chats (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE NO ACTION,
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE ON UPDATE NO ACTION,
    title TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- MESSAGES
CREATE TABLE IF NOT EXISTS  messages (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE ON UPDATE NO ACTION,
    is_user BOOLEAN NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

-- QUIZZES
CREATE TABLE IF NOT EXISTS  quizzes (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    document_id UUID NOT NULL REFERENCES files(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    title TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

-- QUIZ QUESTIONS
CREATE TABLE IF NOT EXISTS  quiz_questions (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    question_no INTEGER NOT NULL,
    quiz_id UUID NOT NULL REFERENCES quizzes(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    question_text TEXT NOT NULL
);

-- QUIZ QUESTION OPTIONS
CREATE TABLE IF NOT EXISTS  quiz_question_options (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    option TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL
);

-- QUIZ ATTEMPTS
CREATE TABLE IF NOT EXISTS quiz_attempts (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    quiz_id UUID NOT NULL REFERENCES quizzes(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    correct_answers INTEGER,
    time_spent_seconds INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL
);

-- USER QUIZ ANSWERS
CREATE TABLE IF NOT EXISTS  user_quiz_answers (
    id UUID PRIMARY KEY UNIQUE NOT NULL,
    attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    user_answer TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL
);

-- INDEXES
CREATE INDEX IF NOT EXISTS user_quiz_answers_index_0 ON user_quiz_answers (attempt_id, question_id);

	`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		panic("Failed to create tables: " + err.Error())
	}
}
