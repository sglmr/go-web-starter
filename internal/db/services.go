package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/sglmr/gowebstart/internal/argon2id"
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
