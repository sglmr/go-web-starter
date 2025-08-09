package data

import (
	"context"
	"errors"
	"testing"
)

func TestAuthenticateLoginService(t *testing.T) {
	t.Parallel()

	// Create a new test database
	database := NewTestDatabase(t, context.Background())
	queries := New(database)

	ctx := context.Background()

	// Create a test user in the database to work with
	testEmail, testPassword := "joe@example.com", "secret"
	testUser, err := queries.CreateUserService(ctx, testEmail, testPassword)
	if err != nil {
		t.Fatalf("could not set up test with a fake user: %v", err)
	}

	// Try to authenticate with a bad email
	_, err = queries.AuthenticateLogin(ctx, "not-a-chance@example.com", testPassword)
	if !errors.Is(err, ErrAuthFailed) {
		t.Errorf("bad email should have failed: %v", err)
	}

	// Try to authenticate with a bad password
	_, err = queries.AuthenticateLogin(ctx, testEmail, "not-today-buddy")
	if !errors.Is(err, ErrAuthFailed) {
		t.Errorf("bad password should have failed: %v", err)
	}

	// Try to authenticate with good credentials... it should work
	user, err := queries.AuthenticateLogin(ctx, testEmail, testPassword)
	switch {
	case err != nil:
		t.Errorf("expected to authenticate user and didn't: %v", err)
	case user.ID != testUser.ID:
		t.Errorf("wrong user returned %v, want %v", user.ID, testUser.ID)
	}
}
