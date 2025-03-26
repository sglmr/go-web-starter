package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/alexedwards/scs/v2"
	"github.com/justinas/nosurf"
	"github.com/sglmr/gowebstart/internal/vcs"
)

//=============================================================================
//	Template Helpers
//=============================================================================

// newTemplateData constructs a map of data to pass into templates
func newTemplateData(r *http.Request, sessionManager *scs.SessionManager) map[string]any {
	messages, ok := sessionManager.Pop(r.Context(), "messages").([]FlashMessage)
	if !ok {
		messages = []FlashMessage{}
	}

	return map[string]any{
		"CSRFToken": nosurf.Token(r),
		"Messages":  messages,
		"Version":   vcs.Version(),
	}
}

//=============================================================================
//	Flash Message functions
//=============================================================================

type contextKey string

const flashMessageKey = "messages"

type FlashMessageLevel string

const (
	// Different FlashMessageLevel types
	LevelSuccess FlashMessageLevel = "success"
	LevelError   FlashMessageLevel = "error"
	LevelWarning FlashMessageLevel = "warning"
	LevelInfo    FlashMessageLevel = "info"
)

type FlashMessage struct {
	Level   FlashMessageLevel
	Message string
}

// putFlashMessage adds a flash message into the session manager
func putFlashMessage(r *http.Request, level FlashMessageLevel, message string, sessionManager *scs.SessionManager) {
	newMessage := FlashMessage{
		Level:   level,
		Message: message,
	}

	// Create a new flashMessageKey context key if one doesn't exist and add the message
	messages, ok := sessionManager.Get(r.Context(), flashMessageKey).([]FlashMessage)
	if !ok {
		sessionManager.Put(r.Context(), flashMessageKey, []FlashMessage{newMessage})
		return
	}

	// Add a flash message to an existing flashMessageKey context key
	messages = append(messages, newMessage)
	sessionManager.Put(r.Context(), flashMessageKey, messages)
}

//=============================================================================
//	Response Helper functions
//=============================================================================

// serverError handles server error http responses.
func serverError(w http.ResponseWriter, r *http.Request, err error, logger *slog.Logger, showTrace bool) {
	// TODO: find some way of reporting the server error
	// app.reportserverError(r, err)

	message := "The server encountered a problem and could not process your request"

	// Display the stack trace on the web page if env is development is on
	if showTrace {
		body := fmt.Sprintf("%s\n\n%s", err, string(debug.Stack()))
		http.Error(w, body, http.StatusInternalServerError)
		return
	}
	logger.Error("server error", "status", http.StatusInternalServerError, "error", err)

	http.Error(w, message, http.StatusInternalServerError)
}

// NotFound handles not found http responses.
func NotFound(w http.ResponseWriter, r *http.Request) {
	message := "The requested resource could not be found"
	http.Error(w, message, http.StatusNotFound)
}

// BadRequest hadles bad request http responses.
func BadRequest(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}
