package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/alexedwards/argon2id"
)

// ChangeUserPasswordService changes the password(hash) for a user.
// It takes the context, email, and new plain-text password as input.
// It returns an error if the password change fails.
func (q *Queries) ChangeUserPasswordService(ctx context.Context, email, password string) error {
	hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	rowCount, err := q.UpdateUserPasswordByEmail(ctx, UpdateUserPasswordByEmailParams{PasswordHash: []byte(hashedPassword), Email: email})
	switch {
	case err != nil:
		return err
	case rowCount == 0:
		return errors.New("no user updated")
	case rowCount > 1:
		return errors.New("multiple users updated")
	}
	return nil
}

// CreateUserService creates a new user in the database with a hashed password.
// It takes the context, email, and plain-text password as input.
// It returns the newly created user if successful, or an error if creation fails.
func (q *Queries) CreateUserService(ctx context.Context, email, password string) (*User, error) {
	hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := q.CreateUser(ctx, CreateUserParams{
		Email:        email,
		PasswordHash: []byte(hashedPassword),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user in database: %w", err)
	}

	return &user, nil
}

var ErrAuthFailed = errors.New("authentication failed")

// AuthenticateLogin authenticates a user's login credentials.
// It takes the context, email, and password as input.
// It returns the authenticated user if successful, or an error if authentication fails
// (e.g., user not found, incorrect password, or other database errors).
func (q *Queries) AuthenticateLogin(ctx context.Context, email, password string) (*User, error) {
	dummyHash, err := argon2id.CreateHash("dummyPassword", argon2id.DefaultParams)
	if err != nil {
		return nil, err
	}

	// Start with a dummy user to ensure a constant time comparison
	user := User{
		PasswordHash: []byte(dummyHash),
	}

	// Query for the user to check if they exist
	foundUser, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		// Return when the error is not sql.ErrNoRows
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		// Continue for an ErrNoRows to avoid timing attacks.
	} else {
		user = foundUser
	}

	// Compare the user's hashed password with the password
	match, err := argon2id.ComparePasswordAndHash(password, string(user.PasswordHash))
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, ErrAuthFailed
	}

	// Return the user
	return &user, nil
}
