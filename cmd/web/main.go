package main

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/sglmr/gowebstart/internal/email"
)

//=============================================================================
// Top level application functions
//=============================================================================

func init() {
	gob.Register(FlashMessage{})
	gob.Register([]FlashMessage{})
}

func main() {
	// Get the background context to pass through the application
	ctx := context.Background()

	// Run the application
	if err := runApp(ctx, os.Stdout, os.Args, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
		return
	}
}

// newServer is a constructor that takes in all dependencies as arguments
func newServer(
	logger *slog.Logger,
	devMode bool,
	mailer email.MailerInterface,
	username, password string,
	wg *sync.WaitGroup,
	sessionManager *scs.SessionManager,
) http.Handler {
	// Create a serve mux
	logger.Debug("creating server")
	mux := http.NewServeMux()

	// Add routes to the ServeMux
	addRoutes(mux, logger, devMode, mailer, username, password, wg, sessionManager)

	// Middleware for all routes
	var handler http.Handler = mux
	handler = recoverPanicMW(handler, logger, devMode)
	handler = secureHeadersMW(handler)
	handler = authenticateMW(sessionManager)(handler)
	handler = sessionManager.LoadAndSave(handler)
	handler = logRequestMW(logger)(handler)

	return handler
}

func runApp(
	ctx context.Context,
	w io.Writer,
	args []string,
	getenv func(string) string,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a waitgroup with 1 item for handling shutdown
	wg := sync.WaitGroup{}
	wg.Add(1)

	// New Flag set
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)

	host := fs.String("host", "0.0.0.0", "Server host")
	port := fs.String("port", "", "Server port")
	devMode := fs.Bool("dev", false, "Development mode. Displays stack trace & more verbose logging")
	username := fs.String("auth-email", getenv("AUTH_EMAIL"), "Email for authentication")
	password := fs.String("auth-password-hash", getenv("AUTH_PASSWORD_HASH"), "Password hash for authentication")
	sendEmail := fs.Bool("send-email", false, "Send live emails")
	smtpHost := fs.String("smtp-host", getenv("SMTP_HOST"), "Email smtp host")
	smtpPortString := fs.String("smtp-port", getenv("SMTP_PORT"), "Email smtp port")
	smtpUsername := fs.String("smtp-username", getenv("SMTP_USERNAME"), "Email smtp username")
	smtpPassword := fs.String("smtp-password", getenv("SMTP_PASSWORD"), "Email smtp password")
	smtpFrom := fs.String("smtp-from", getenv("SMTP_EMAIL"), "Email smtp Sender")

	// Parse the flags
	err := fs.Parse(args[1:])
	if err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	// Parse the smtp port
	var smtpPort int
	switch {
	case *smtpPortString == "" && *devMode:
		smtpPort = 0
	default:
		smtpPort, err = strconv.Atoi(*smtpPortString)
		if err != nil {
			return fmt.Errorf("error parsing smtpPort: %w", err)
		}
	}

	// Get port from environment
	if *port == "" {
		*port = getenv("PORT")
	}
	if *port == "" {
		*port = "8000"
	}

	// Create a new logger
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)
	logger := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: logLevel,
	}))
	if *devMode {
		logLevel.Set(slog.LevelDebug)
	}

	// Create a mailer for sending emails
	var mailer email.MailerInterface
	switch *sendEmail {
	case true:
		// Configure a mailer to send real emails
		mailer, err = email.NewMailer(*smtpHost, smtpPort, *smtpUsername, *smtpPassword, *smtpFrom)
		if err != nil {
			logger.Error("smtp configuration error", "error", err)
			return fmt.Errorf("smtp mailer setup failed: %w", err)
		}
	default:
		mailer = email.NewLogMailer(logger)
	}

	// Session manager configuration
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour

	// Set up router
	srv := newServer(logger, *devMode, mailer, *username, *password, &wg, sessionManager)

	// Configure an http server
	httpServer := &http.Server{
		Addr:         net.JoinHostPort(*host, *port),
		Handler:      srv,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// This pattern is starts a server background while the main program continues with other tasks.
	// The main program can later stop the server using httpServer.Shutdown().
	go func() {
		logger.Info("application running (press ctrl+C to quit)", "address", fmt.Sprintf("http://%s", httpServer.Addr))

		// httpServer.ListenAndServe() begins listening for HTTP requests
		// This method blocks (runs forever) until the server is shut down
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Print an error if any error other than http.ErrServerclosed shows up
			logger.Error("listen and serve error", "error", err)
			// Send SIGTERM to self to shutdown the application
			p, _ := os.FindProcess(os.Getpid())
			p.Signal(syscall.SIGTERM)
		}
	}()

	// Start a goroutine to handle server shutdown
	go func() {
		// The waitgroup counter will decrement and signal complete at
		// the end of this function
		defer wg.Done()

		// This blocks the goroutine until the ctx context is cancelled
		<-ctx.Done()
		logger.Info("waiting for application to shutdown")

		// Create an empty context for the shutdown process with a 10 second timer
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Initiate a graceful shutdown of the server and handle any errors
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("error shutting down http server: %s\n", "error", err)
		}
	}()
	// Makes the goroutine wait until shutdown starts
	wg.Wait()
	logger.Info("application shutdown complete")
	return nil
}

// backgroundTask executes a function in a background goroutine with proper error handling.
func backgroundTask(wg *sync.WaitGroup, logger *slog.Logger, fn func() error) {
	// Increment waitgroup to track whether this background task is complete or not
	wg.Add(1)

	// Launch a goroutine to run the task in
	go func() {
		// decrement the waitgroup after the task completes
		defer wg.Done()

		// Get the name of the function
		funcName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()

		// Recover any panics in the task function so that
		// a panic doesn't kill the whole application
		defer func() {
			err := recover()
			if err != nil {
				logger.Error("task", "name", funcName, "error", fmt.Errorf("%s", err))
			}
		}()

		// Execute the provided function, logging any errors
		err := fn()
		if err != nil {
			logger.Error("task", "name", funcName, "error", err)
		}
	}()
}
