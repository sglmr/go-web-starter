package web

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sglmr/gowebstart/internal/argon2id"
	"github.com/sglmr/gowebstart/internal/db"
)

var ErrAuthFailed = errors.New("authentication failed")

// AuthenticateLogin authenticates a user's login credentials.
// It takes the context, email, and password as input.
// It returns the authenticated user if successful, or an error if authentication fails
// (e.g., user not found, incorrect password, or other database errors).
func (app *Application) AuthenticateLogin(ctx context.Context, email, password string) (*db.User, error) {
	dummyHash, err := argon2id.CreateHash("dummyPassword", argon2id.DefaultParams)
	if err != nil {
		return nil, err
	}

	// Start with a dummy user to ensure a constant time comparison
	user := db.User{
		PasswordHash: []byte(dummyHash),
	}

	// Query for the user to check if they exist
	foundUser, err := app.Queries.GetUserByEmail(ctx, email)
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
