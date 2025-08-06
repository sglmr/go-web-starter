package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sglmr/gowebstart/internal/db"
)

func main() {
	// Flags for "create-user" command to create a new user
	createUserCmd := flag.NewFlagSet("create-user", flag.ExitOnError)
	emailPtr := createUserCmd.String("email", "", "Email for the new user")
	passwordPtr := createUserCmd.String("password", "", "Password for the new user")
	// createUserDsn := createUserCmd.String("dsn", "", "Database connection path")

	if len(os.Args) < 2 {
		fmt.Println("expected 'create-user' subcommand")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create-user":
		createUserCmd.Parse(os.Args[2:])
		if *emailPtr == "" || *passwordPtr == "" {
			createUserCmd.PrintDefaults()
			os.Exit(1)
		}
		createUser(*emailPtr, *passwordPtr)
	default:
		fmt.Println("expected 'create-user' subcommand")
		os.Exit(1)
	}
}

func createUser(email, password string) {
	dsn := os.Getenv("WEB_DSN")
	if dsn == "" {
		log.Fatal("WEB_DSN environment variable not set")
	}

	database, err := db.NewDatabaseConnection(dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()
	queries := db.New(database)

	user, err := queries.CreateUserService(context.Background(), email, password)
	if err != nil {
		log.Fatalf("failed to create user: %v", err)
	}

	fmt.Printf("User created successfully with ID: %d and email: %s\n", user.ID, user.Email)
}
