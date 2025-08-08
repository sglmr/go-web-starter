package web

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/alexedwards/scs/v2"
	"github.com/justinas/nosurf"
	"github.com/sglmr/gowebstart/internal/vcs"
)

type contextKey string

//	Template functions

// newTemplateData constructs a map of data to pass into templates
func newTemplateData(r *http.Request, SessionManager *scs.SessionManager) map[string]any {
	messages, ok := SessionManager.Pop(r.Context(), "messages").([]FlashMessage)
	if !ok {
		messages = []FlashMessage{}
	}

	return map[string]any{
		"CSRFToken":       nosurf.Token(r),
		"IsAuthenticated": isAuthenticated(r),
		"Messages":        messages,
		"UrlPath":         r.URL.Path,
		"Version":         vcs.Version(),
	}
}

//	FlashMessage functions

const flashMessageKey = "messages"

type flashLevel string

const (
	// Different flashLevel types
	flashInfo    flashLevel = "info"
	flashSuccess flashLevel = "success"
	flashWarning flashLevel = "warning"
	flashError   flashLevel = "error"
)

type FlashMessage struct {
	Level   flashLevel
	Message string
}

// Flash adds a flash message to the session
func (app *Application) Flash(r *http.Request, level flashLevel, message string) {
	newMessage := FlashMessage{
		Level:   level,
		Message: message,
	}

	// Create a new flashMessageKey context key if one doesn't exist and add the message
	messages, ok := app.SessionManager.Get(r.Context(), flashMessageKey).([]FlashMessage)
	if !ok {
		app.SessionManager.Put(r.Context(), flashMessageKey, []FlashMessage{newMessage})
		return
	}

	// Add a flash message to an existing flashMessageKey context key
	messages = append(messages, newMessage)
	app.SessionManager.Put(r.Context(), flashMessageKey, messages)
}

//	Response Helper functions

// serverError handles server error http responses.
func (app *Application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	// TODO: find some way of reporting the server error
	// app.reportserverError(r, err)

	message := "The server encountered a problem and could not process your request"

	// Display the stack trace on the web page if env is development is on
	if app.DevMode {
		body := fmt.Sprintf("%s\n\n%s", err, string(debug.Stack()))
		http.Error(w, body, http.StatusInternalServerError)
		return
	}
	app.Log.Error("server error", "status", http.StatusInternalServerError, "error", err)

	http.Error(w, message, http.StatusInternalServerError)
}

// clientError returns a user/client error response
func clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

// Authentication Helpers

const (
	isAuthenticatedContextKey = contextKey("isAuthenticated")
	isAnonyousContextKey      = contextKey("isAnonymous")
)

// isAuthenticated returns true when a user is authenticated. The function checks the
// request context for a isAuthenticatedContextKey value
func isAuthenticated(r *http.Request) bool {
	isAuthenticated, ok := r.Context().Value(isAuthenticatedContextKey).(bool)
	if !ok {
		return false
	}
	return isAuthenticated
}
