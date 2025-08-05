package db

import (
	"database/sql"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// DB holds the database connections and a mutex for writing.
type DB struct {
	writeDB *sql.DB
	readDB  *sql.DB
	mu      sync.Mutex // Mutex to lock for writes
}

// NewDatabase returns a new DB instance.
// We name it NewDatabase to avoid a conflict with the sqlc-generated New function.
func NewDatabase(path string) (*DB, error) {
	// Create a write db
	writeDB, err := sql.Open("sqlite3", path+"?_journal=WAL")
	if err != nil {
		return nil, err
	}

	// Create a read db
	readDB, err := sql.Open("sqlite3", path+"?_journal=WAL")
	if err != nil {
		return nil, err
	}

	return &DB{
		writeDB: writeDB,
		readDB:  readDB,
	}, nil
}

// Close closes the database connections.
func (db *DB) Close() {
	// Lock the mutex to ensure no new writes start while we are closing.
	db.mu.Lock()
	defer db.mu.Unlock()

	db.writeDB.Close()
	db.readDB.Close()
}

// Write returns the write database connection.
func (db *DB) Write() *sql.DB {
	return db.writeDB
}

// Read returns the read database connection.
func (db *DB) Read() *sql.DB {
	return db.readDB
}

// Lock locks the mutex for writing.
func (db *DB) Lock() {
	db.mu.Lock()
}

// Unlock unlocks the mutex for writing.
func (db *DB) Unlock() {
	db.mu.Unlock()
}

// NewTestDatabase creates a new in-memory sqlite database for testing.
func NewTestDatabase(t *testing.T) (*DB, error) {
	// Create an in-memory SQLite database
	db, err := NewDatabase("file::memory:?cache=shared")
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

	return db, nil
}
