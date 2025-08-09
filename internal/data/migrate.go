package data

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// MigrateUp runs all up migrations.
func MigrateUp(dbPath string) error {
	// use the embedded file system
	source, err := iofs.New(EmbeddedFiles, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance(
		"iofs",
		source,
		fmt.Sprintf("sqlite3://%s", dbPath),
	)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// MigrateDown runs all down migrations.
func MigrateDown(dbPath string) error {
	// use the embedded file system
	source, err := iofs.New(EmbeddedFiles, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance(
		"iofs",
		source,
		fmt.Sprintf("sqlite3://%s", dbPath),
	)
	if err != nil {
		return err
	}
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
