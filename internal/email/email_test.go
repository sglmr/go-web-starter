package email

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLogMailer_Send(t *testing.T) {
	t.Parallel()

	// Create a buffer to capture log output
	var logBuffer bytes.Buffer

	// Create a test logger that writes to our buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create the LogMailer with our test logger
	logMailer := NewLogMailer(logger)

	// Test data
	recipient := "test@example.com"
	replyTo := "reply@example.com"
	testData := map[string]string{"name": "Test User", "message": "Hello World"}
	patterns := []string{"welcome.tmpl", "notification.tmpl"}

	// Call the Send method
	err := logMailer.Send(recipient, replyTo, testData, patterns...)
	if err != nil {
		t.Fatalf("error sending log mail: %v", err)
	}

	// Assert the log StringIn the expected information
	logOutput := logBuffer.String()

	testCases := []struct {
		name string // name of the test & substring that we want
	}{
		{name: "send email"},
		{name: "recipient=test@example.com"},
		{name: "welcome.tmpl"},
		{name: "Hello World"},
		{name: "notification.tmpl"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(logOutput, tc.name) {
				t.Errorf("log output missing: %q, output: %v", tc.name, logOutput)
			}
		})
	}
}

// TestLogMailerImplementsInterface ensures that LogMailer correctly implements MailerInterface
func TestLogMailerImplementsInterface(t *testing.T) {
	t.Parallel()
	// This is a compile time check that LogMailer implements the MailerInterface
	var _ MailerInterface = (*LogMailer)(nil)
}

// TestMailerImplementsInterface ensures that Mailer correctly implements MailerInterface
func TestMailerImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ MailerInterface = (*Mailer)(nil)
}
