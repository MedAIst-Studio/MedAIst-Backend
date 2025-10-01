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
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		fullname VARCHAR(255),
		role VARCHAR(10) NOT NULL CHECK (role IN ('USER', 'ADMIN')),
		password VARCHAR(200) NOT NULL,
		email VARCHAR(200) NOT NULL UNIQUE,
		phone VARCHAR(20),
		is_email_verified BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS carts (
		id UUID PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id),
		status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS donation_types(
		id UUID PRIMARY KEY,
		code TEXT NOT NULL,
		language TEXT NOT NULL,
		title TEXT,
		UNIQUE (code, language)
	);

	CREATE TABLE IF NOT EXISTS donations (
		id UUID PRIMARY KEY,
		cart_id UUID NOT NULL REFERENCES carts(id),
		donation_type_code TEXT NOT NULL,
		amount DECIMAL(10,2) NOT NULL,
		currency VARCHAR(10) NOT NULL
	);


	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id UUID PRIMARY KEY,
		user_id UUID NOT NULL UNIQUE REFERENCES users(id),
		token TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_carts_user_id ON carts(user_id);
	CREATE INDEX IF NOT EXISTS idx_donations_cart_id ON donations(cart_id);
	CREATE INDEX IF NOT EXISTS idx_donations_type_code ON donations(donation_type_code);
	`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		panic("Failed to create tables: " + err.Error())
	}
}
