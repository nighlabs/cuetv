package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// New opens (or creates) a SQLite database at dbPath with WAL journal mode and
// foreign key enforcement enabled. It verifies connectivity, runs all pending
// migrations, and returns the ready-to-use connection.
func New(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	slog.Info("database initialized", "path", dbPath)
	return db, nil
}

// NewInMemory opens an in-memory SQLite database with foreign key enforcement
// and runs all migrations. It is intended for use in tests where a temporary,
// disposable database is needed.
func NewInMemory() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening in-memory database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// runMigrations reads all .sql files from the embedded migrations filesystem
// and executes them sequentially against db. Files are processed in
// lexicographic order, so migration filenames should use a numeric prefix
// (e.g. 001_create_sessions.sql) to ensure correct ordering.
func runMigrations(db *sql.DB) error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	for _, entry := range entries {
		content, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", entry.Name(), err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("executing migration %s: %w", entry.Name(), err)
		}

		slog.Info("migration applied", "file", entry.Name())
	}

	return nil
}
