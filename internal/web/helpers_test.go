package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sglmr/gowebstart/internal/data"
	"github.com/sglmr/gowebstart/internal/render"
	"github.com/sglmr/gowebstart/internal/vcs"
)

func TestNewTemplateData(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	// Create a temporary http handler to be wrapped with the scs LoadAndSave middleware.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Flash a test message.
		templateData := newTemplateData(r, app.SessionManager)

		if _, ok := templateData["CSRFToken"]; !ok {
			t.Error("CSRFToken not found in template data")
		}
		if _, ok := templateData["IsAuthenticated"]; !ok {
			t.Error("IsAuthenticated not found in template data")
		}
		if _, ok := templateData["Messages"]; !ok {
			t.Error("Messages not found in template data")
		}
		urlPath, ok := templateData["UrlPath"]
		if !ok {
			t.Error("UrlPath not found in template data")
		}
		if got, want := urlPath, "/"; got != want {
			t.Errorf("got url path %q, wanted %q", got, want)
		}

		version, ok := templateData["Version"]
		if !ok {
			t.Error("Version not found in template data")
		}
		if got, want := version, vcs.Version(); got != want {
			t.Errorf("got version %q, wanted %q", got, want)
		}
	})

	// 2. The handler is wrapped with the LoadAndSave middleware.
	mux := app.SessionManager.LoadAndSave(handler)

	// 3. We create a request and a "recorder" to simulate a real request cycle.
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// 4. We serve the request to the middleware-wrapped handler.
	mux.ServeHTTP(rr, r)
}

func TestFlashMessage(t *testing.T) {
	t.Parallel()

	testString := "Testing time!"

	app := newTestApplication(t)

	// Create a temporary http handler to be wrapped with the scs LoadAndSave middleware.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Flash a test message.
		app.Flash(r, flashInfo, testString)

		// Assertions are now made within the handler, where the session context is active.
		messages, ok := app.SessionManager.Get(r.Context(), "messages").([]FlashMessage)
		if !ok {
			t.Fatal("no messages in session")
		}
		if got, want := len(messages), 1; got != want {
			t.Errorf("got %v message(s), expected %v", got, want)
		}
		if got, want := messages[0].Message, testString; got != want {
			t.Errorf("got %q, wanted %q", got, want)
		}
		if got, want := messages[0].Level, flashInfo; got != want {
			t.Errorf("got level %v, wanted level %v", got, want)
		}

		// Serve up the home page template so that we can test if the response body
		// contains the flash message.
		data := newTemplateData(r, app.SessionManager)
		if err := render.Page(w, http.StatusOK, data, "home.tmpl"); err != nil {
			app.serverError(w, r, err)
			return
		}
	})

	// 2. The handler is wrapped with the LoadAndSave middleware.
	mux := app.SessionManager.LoadAndSave(handler)

	// 3. We create a request and a "recorder" to simulate a real request cycle.
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// 4. We serve the request to the middleware-wrapped handler.
	mux.ServeHTTP(rr, r)

	// Check if the response body contains the flash message.
	if !strings.Contains(rr.Body.String(), testString) {
		t.Errorf("missing flash message in %v", rr.Body)
	}
}

func TestServerError(t *testing.T) {
	t.Parallel()

	t.Run("test serverError in DevMode", func(t *testing.T) {
		app := newTestApplication(t)
		app.DevMode = true

		r, _ := http.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		app.serverError(rr, r, errors.New("faillll"))

		// Test the response status code
		if got, want := rr.Code, http.StatusInternalServerError; got != want {
			t.Errorf("got status code %v, wanted %v", got, want)
		}

		// Test the body contains the stack trace in dev mode
		if !strings.Contains(rr.Body.String(), "runtime/debug.Stack()") {
			t.Fatalf("missing stacktrace from body %v", rr.Body.String())
		}
	})

	t.Run("test serverError NOT in DevMode", func(t *testing.T) {
		app := newTestApplication(t)
		app.DevMode = false

		r, _ := http.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		app.serverError(rr, r, errors.New("faillll"))

		// Test the response status code
		if got, want := rr.Code, http.StatusInternalServerError; got != want {
			t.Errorf("got status code %v, wanted %v", got, want)
		}

		// Test the body does not contain the stack trace
		if strings.Contains(rr.Body.String(), "runtime/debug.Stack()") {
			t.Fatalf("stacktrace should't be in the body %v", rr.Body.String())
		}

		// Test the body content contains "The server encountered a problem..."
		if !strings.Contains(rr.Body.String(), "server encountered a problem") {
			t.Fatalf("missing %q from %v", "server encountered a problem", rr.Body.String())
		}
	})
}

func TestClientError(t *testing.T) {
	t.Parallel()

	// 1. Define the test cases in a "table" (a slice of structs).
	testCases := []struct {
		name            string // A description of the test case
		statusCode      int    // The HTTP status code to test
		expectedSnippet string // The text expected in the response body
	}{
		{
			name:            "Not Found",
			statusCode:      http.StatusNotFound,
			expectedSnippet: "Not Found",
		},
		{
			name:            "Bad Request",
			statusCode:      http.StatusBadRequest,
			expectedSnippet: "Bad Request",
		},
		{
			name:            "Unauthorized",
			statusCode:      http.StatusUnauthorized,
			expectedSnippet: "Unauthorized",
		},
		{
			name:            "Method Not Allowed",
			statusCode:      http.StatusMethodNotAllowed,
			expectedSnippet: "Method Not Allowed",
		},
	}

	// 2. Loop over the test cases.
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh ResponseRecorder for each test case for isolation.
			rr := httptest.NewRecorder()

			// Make a client error
			clientError(rr, tc.statusCode)

			// Assertion on status code
			if got, want := rr.Code, tc.statusCode; got != want {
				t.Errorf("got status %v, wanted %v", got, want)
			}

			// Assertion on page contents
			if !strings.Contains(rr.Body.String(), tc.expectedSnippet) {
				t.Errorf("missing %q in body: %v", tc.expectedSnippet, rr.Body.String())
			}
		})
	}
}

func TestContextSetUserGetuser(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	user := data.User{
		ID:    1,
		Email: "test@example.com",
	}

	r = app.contextSetUser(r, user)
	gotUser := app.contextGetUser(r)

	if got, want := gotUser.ID, user.ID; got != want {
		t.Errorf("got ID %v, wanted %v", got, want)
	}

	if got, want := gotUser.Email, user.Email; got != want {
		t.Errorf("got email %q, wanted %q", got, want)
	}
}

func TestContextGetUserMissingUser(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)
	r, _ := http.NewRequest(http.MethodGet, "/", nil)

	assertPanicsWithValue(t, "missing user value in request context", func() { app.contextGetUser(r) })
}

func addIsAuthenticatedContext(t *testing.T, ctx context.Context, isAuthenticated bool) context.Context {
	t.Helper()
	return context.WithValue(ctx, isAuthenticatedContextKey, isAuthenticated)
}

func TestIsAuthenticated(t *testing.T) {
	t.Run("is authenticated", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		ctx := r.Context()
		r = r.WithContext(addIsAuthenticatedContext(t, ctx, true))
		got := isAuthenticated(r)
		if !got {
			t.Errorf("isAuthenticated should have been 'true'")
		}
	})

	t.Run("is not authenticated", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		ctx := r.Context()
		r = r.WithContext(addIsAuthenticatedContext(t, ctx, false))
		got := isAuthenticated(r)

		if got {
			t.Errorf("isAuthenticated should have been 'false'")
		}
	})

	t.Run("is not in context", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		got := isAuthenticated(r)

		if got {
			t.Errorf("isAuthenticated should have been 'false'")
		}
	})
}
