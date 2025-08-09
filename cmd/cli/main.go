package main

import (
	"context"
	"fmt"
	"log"

	"github.com/alecthomas/kong"
	"github.com/sglmr/gowebstart/internal/data"
)

var CLI struct {
	CreateUser struct {
		Email    string `help:"Email for the new user." required:""`
		Password string `help:"Password for the new user." required:""`
		DSN      string `help:"Path to database." required:""`
	} `cmd:"" help:"Creates a new user."`
	ChangePassword struct {
		Email    string `help:"Email for the user." required:""`
		Password string `help:"New password for the user." required:""`
		DSN      string `help:"Path to database." required:""`
	} `cmd:"" help:"Changes a user's password."`
}

func main() {
	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	case "create-user":
		createUser(CLI.CreateUser.DSN, CLI.CreateUser.Email, CLI.CreateUser.Password)
	case "change-password":
		changePassword(CLI.ChangePassword.DSN, CLI.ChangePassword.Email, CLI.ChangePassword.Password)
	default:
		panic(ctx.Command())
	}
}

func createUser(dsn, email, password string) {
	// Connect to the database
	database, err := data.NewDatabaseConnection(dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()
	queries := data.New(database)

	// Create a new user
	user, err := queries.CreateUserService(context.Background(), email, password)
	if err != nil {
		log.Fatalf("failed to create user: %v", err)
	}

	// Print success message.
	fmt.Printf("User created successfully with ID: %d and email: %s", user.ID, user.Email)
}

func changePassword(dsn, email, password string) {
	// Connect to the database
	database, err := data.NewDatabaseConnection(dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()
	queries := data.New(database)

	// Change password
	err = queries.ChangeUserPasswordService(context.Background(), email, password)
	if err != nil {
		log.Fatalf("failed to change password: %v", err)
	}

	fmt.Printf("Password for user '%s' changed successfully.", email)
}
