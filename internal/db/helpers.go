package db

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// NewDatabase returns a new *sql.DB instance.
// We name it NewDatabase to avoid a conflict with the sqlc-generated New function.
func NewDatabaseConnection(path string) (*sql.DB, error) {
	var dsn string

	// Set sqlite pragma options as a query string if there isn't already a query string
	if !strings.Contains(path, "?") {
		options := url.Values{}
		options.Set("_journal_mode", "WAL")
		options.Set("_synchronous", "NORMAL")
		options.Set("_mmap_size", "134217728")
		options.Set("_journal_size_limit", "67108864")
		options.Set("_cache_size", "-2000")
		options.Set("_busy_timeout", "5000") // in ms; 5000 = 5 seconds
		options.Set("_txlock", "immediate")
		options.Set("_foreign_keys", "true")

		// The connection string to the database
		dsn = path + "?" + options.Encode()
	} else {
		dsn = path
	}

	// Create a database connection
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return database, nil
}

// NewTestDatabase creates a new in-memory sqlite database for testing.
func NewTestDatabase(t *testing.T, ctx context.Context) (*sql.DB, error) {
	// Create an in-memory SQLite database
	db, err := NewDatabaseConnection("file::memory:?cache=shared")
	if err != nil {
		return nil, err
	}

	// Register a function to automatically cleanup after
	// this function's caller is finished.
	t.Cleanup(func() {
		// This would perform down migrations
		// and any cleanup activities if we weren't using an in-memory database.
	})

	// Run migrations
	// Note: The dbPath for migrations still needs to be the connection string.
	if err := MigrateUp("file::memory:?cache=shared"); err != nil {
		return nil, err
	}

	// Seed the test database with data
	loadTestData(t, ctx, db)

	return db, nil
}

func loadTestData(t *testing.T, ctx context.Context, db *sql.DB) error {
	queries := New(db)

	// Create admin@example.com user
	_, err := queries.CreateUserService(ctx, "admin@example.com", "password")
	if err != nil {
		t.Error("could not create test user : ", err)
	}

	return nil
}
