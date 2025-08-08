package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
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
func NewTestDatabase(t *testing.T, ctx context.Context) *sql.DB {
	// This connection string will be unique for each test that creates a database.
	tempFile, err := os.CreateTemp("", fmt.Sprintf("%s_*", t.Name()))
	if err != nil {
		t.Fatalf("could not create temp databse file")
	}

	dsn := fmt.Sprintf("%s?mode=memory&cache=shared", tempFile.Name())

	// Create an in-memory SQLite database
	db, err := NewDatabaseConnection(dsn)
	if err != nil {
		t.Fatalf("test database connection failed: %v", err)
	}

	// Register a function to automatically cleanup after
	// this function's caller is finished.
	t.Cleanup(func() {
		// Perform down migration. The down migraton
		// should delete any test data that was created.
		if err := MigrateDown(dsn); err != nil {
			t.Fatalf("could not perform down migration")
		}

		// Delete the temp file that was created
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Fatalf("could not delete test database file")
		}
	})

	// Run up migrations
	// Note: The dbPath for migrations still needs to be the connection string.
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("test database migration up failed: %v", err)
	}

	// Seed the test database with data
	loadTestData(t, ctx, db)

	return db
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
