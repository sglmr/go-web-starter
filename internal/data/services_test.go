package data

import (
	"context"
	"errors"
	"testing"

	"github.com/alexedwards/argon2id"
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

func TestChangeUserPasswordService(t *testing.T) {
	t.Parallel()

	testEmail := "admin@example.com"
	newPassword := "newPassword"
	ctx := context.Background()

	database := NewTestDatabase(t, ctx)
	queries := New(database)

	user, err := queries.GetUserByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("could not query for a user with %q", testEmail)
	}

	// Check that the starting password for the user is a known value
	// before changing it.
	match, err := argon2id.ComparePasswordAndHash("password", string(user.PasswordHash))
	if err != nil {
		t.Fatalf("error comparing password and hash: %v", err)
	}
	if !match {
		t.Errorf("the starting password for the user did not work")
	}

	// Try to change the user's password
	err = queries.ChangeUserPasswordService(ctx, testEmail, newPassword)
	if err != nil {
		t.Fatalf("reset password error: %v", err)
	}

	// Refresh the User type
	user, err = queries.GetUserByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("could not query for a user after changing their password with %q: %v", testEmail, err)
	}

	// Check that the password reset worked
	match, err = argon2id.ComparePasswordAndHash(newPassword, string(user.PasswordHash))
	if err != nil {
		t.Fatalf("could not compare the new password: %v", err)
	}
	if !match {
		t.Fatalf("the user's password didn't match the new password after using the ChangeUserPasswordService")
	}
}
