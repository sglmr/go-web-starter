package web

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"gotest.tools/assert"
)

func TestStaticFileSystem_Open(t *testing.T) {
	t.Parallel()

	// 1. Define the mock filesystem for our tests using fstest.MapFS.
	// This map represents the file structure we want to test against.
	testFS := fstest.MapFS{
		// A valid file that should be served.
		"static/css/style.css": {Data: []byte("body {}")},
		// A directory that contains an index.html file.
		"static/js/index.html": {Data: []byte("<html></html>")},
		// A file that is outside the allowed 'static' path.
		"private/secret.txt": {Data: []byte("secret")},
		// A directory that does NOT contain an index.html, to test directory listing prevention.
		"static/img": {Mode: fs.ModeDir},
	}

	// 2. Instantiate the code you want to test.
	sfs := staticFileSystem{fs: testFS}

	// 3. Define the test cases.
	testCases := []struct {
		name    string // The name of the test case
		path    string // The file path to request
		wantErr error  // The expected error, if any
		isDir   bool   // Whether we expect the result to be a directory
	}{
		{
			name:    "should open a valid file",
			path:    "static/css/style.css",
			wantErr: nil,
			isDir:   false,
		},
		{
			name:    "should prevent access to files outside 'static' path",
			path:    "private/secret.txt",
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "should return an error for a non-existent file",
			path:    "static/nonexistent.file",
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "should allow access to a directory that contains index.html",
			path:    "static/js",
			wantErr: nil,
			isDir:   true,
		},
		{
			name:    "should prevent access to a directory without index.html",
			path:    "static/img",
			wantErr: fs.ErrNotExist,
		},
	}

	// 4. Run the sub-tests.
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := sfs.Open(tc.path)
			if file != nil {
				defer file.Close()
			}

			// Check if the returned error matches the expected error.
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("sfs.Open(%q) error = %v; want %v", tc.path, err, tc.wantErr)
			}

			// If we didn't expect an error, check that the file is not nil.
			if tc.wantErr == nil && file == nil {
				t.Errorf("sfs.Open(%q) expected a file handle but got nil", tc.path)
			}
		})
	}
}

func TestSecureHeadersMW(t *testing.T) {
	t.Parallel()

	// Initialize a new httptest.ResponseRecorder and dummy http.Request.
	rr := httptest.NewRecorder()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock HTTP handler that we can pass to our SecureHeadersMW
	// middleware, which writes a 200 status code and an "OK" response body.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Pass the mock HTTP handler to the SecureHeadersMW middleware.
	// Call ServeHTTP to execute it.
	secureHeadersMW(next).ServeHTTP(rr, r)

	// Get the results of the test
	rs := rr.Result()

	// Check that the middleware has correctly set the Referrer-Policy
	// header on the response.
	want := "origin-when-cross-origin"
	assert.Equal(t, rs.Header.Get("Referrer-Policy"), want)

	// Check that the middleware has correctly set the X-Content-Type-Options
	// header on the response.
	want = "nosniff"
	assert.Equal(t, rs.Header.Get("X-Content-Type-Options"), want)

	// Check that the middleware has correctly set the X-Frame-Options header
	// on the response.
	want = "deny"
	assert.Equal(t, rs.Header.Get("X-Frame-Options"), want)

	// Check that the middleware has correctly set the X-XSS-Protection header
	// on the response
	want = "0"
	assert.Equal(t, rs.Header.Get("X-XSS-Protection"), want)

	// Check that the middleware has correctly called the next handler in line
	// and the response status code and body are as expected.
	assert.Equal(t, rs.StatusCode, http.StatusOK)

	defer rs.Body.Close()
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.TrimSpace(body)

	assert.Equal(t, string(body), "OK")
}

func TestRecoverPanicMW(t *testing.T) {
	t.Parallel()

	// Create a test application
	logBuffer := bytes.Buffer{}
	testLogger := slog.New(slog.NewTextHandler(&logBuffer, nil))
	testApp := Application{
		Log: testLogger,
	}

	// Initialize a new httptest.ResponseRecorder and dummy http.Request.
	rr := httptest.NewRecorder()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock HTTP handler that we can pass to our RecoverPanicMW
	// middleware, which creates a panic
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("Help!")
	})

	// Pass the mock HTTP handler to the RecoverPanicMW middleware.
	// Call ServeHTTP to execute it.
	testApp.recoverPanicMW(next).ServeHTTP(rr, r)

	// Get the results of the test
	rs := rr.Result()

	// Check that the middleware has correctly called the next handler in line
	// and the response status code and body are as expected.
	assert.Equal(t, rs.StatusCode, http.StatusInternalServerError)

	defer rs.Body.Close()
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.TrimSpace(body)

	// Check the body contents
	if got, want := "The server encountered a problem and could not process your request", string(body); got != want {
		t.Errorf("response body missing %q", got)
	}

	// Check the status code
	if got, want := rs.StatusCode, http.StatusInternalServerError; got != want {
		t.Errorf("got response %v, wanted %v", got, want)
	}

	// Check the log message
	logMsg := logBuffer.String()

	if !strings.Contains(logMsg, "level=ERROR") {
		t.Errorf("recover panic WM missing log level")
	}
	if !strings.Contains(logMsg, "status=500") {
		t.Errorf("recover panic WM missing status code")
	}
	if !strings.Contains(logMsg, "error=Help!") {
		t.Errorf("recover panic WM missing error message")
	}
}

// TestTrailingSlashMiddleware tests the middleware for redirecting and passing requests.
func TestTrailingSlashMiddleware(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		path               string
		expectedStatusCode int
		expectedLocation   string
		shouldCallNext     bool
	}{
		{
			name:               "Redirect path without query",
			path:               "/foo",
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "/foo/",
			shouldCallNext:     false,
		},
		{
			name:               "Redirect path with query",
			path:               "/foo?bar=baz&id=123",
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "/foo/?bar=baz&id=123",
			shouldCallNext:     false,
		},
		{
			name:               "Do not redirect path with trailing slash",
			path:               "/foo/",
			expectedStatusCode: http.StatusOK,
			expectedLocation:   "",
			shouldCallNext:     true,
		},
		{
			name:               "Do not redirect static path",
			path:               "/static/css/style.css",
			expectedStatusCode: http.StatusOK,
			expectedLocation:   "",
			shouldCallNext:     true,
		},
		{
			name:               "Do not redirect root path",
			path:               "/",
			expectedStatusCode: http.StatusOK,
			expectedLocation:   "",
			shouldCallNext:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock request to the specified path.
			req := httptest.NewRequest("GET", tc.path, nil)

			// Create a response recorder to capture the response.
			rr := httptest.NewRecorder()

			// Create a mock next handler that sets a flag if it's called.
			nextCalled := false
			mockNextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
			})

			// Create the middleware instance with our mock next handler.
			middleware := trailingSlashMiddleware(mockNextHandler)

			// Serve the request through the middleware.
			middleware.ServeHTTP(rr, req)

			// Check if the status code is what we expect.
			if got, want := rr.Code, tc.expectedStatusCode; got != want {
				t.Errorf("got status code %d, wanted %d", got, want)
			}

			// Check the Location header for redirects.
			if got, want := rr.Header().Get("Location"), tc.expectedLocation; got != want {
				t.Errorf("got loction header %q, wanted %q", got, want)
			}

			// Check if the next handler was called when it should have been.
			if got, want := nextCalled, tc.shouldCallNext; got != want {
				t.Errorf("got next handler: %v, wanted: %v", got, want)
			}
		})
	}
}

// TestRedirectWithSlash uses a table of test cases to check the redirectWithSlash function.
func TestRedirectWithSlash(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Root path",
			path:     "/",
			expected: false,
		},
		{
			name:     "Path with trailing slash",
			path:     "/foo/",
			expected: false,
		},
		{
			name:     "Static path prefix",
			path:     "/static/js/main.js",
			expected: false,
		},
		{
			name:     "Static path with trailing slash",
			path:     "/static/",
			expected: false,
		},
		{
			name:     "Simple path needing a slash",
			path:     "/foo",
			expected: true,
		},
		{
			name:     "Path with multiple segments needing a slash",
			path:     "/foo/bar",
			expected: true,
		},
		{
			name:     "Path that is just /static",
			path:     "/static",
			expected: true,
		},
		{
			name:     "Empty path",
			path:     "",
			expected: true,
		},
	}

	// Iterate over the test cases.
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := redirectWithSlash(tc.path), tc.expected; got != want {
				t.Errorf("got '%v', want '%v'", got, want)
			}
		})
	}
}

func TestCacheControlMW(t *testing.T) {
	t.Parallel()

	// --- 1. Setup ---

	// Create a mock "next" handler that the middleware will call.
	// This confirms that the middleware chain isn't broken.
	mockNextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Set a status code
	})

	// Define the max-age value for this test.
	testMaxAge := "3600"

	// Create an instance of our middleware.
	middleware := cacheControlMW(testMaxAge)

	// Wrap our mock handler with the middleware. This is the handler we will test.
	handlerToTest := middleware(mockNextHandler)

	// Create a new mock HTTP request.
	req := httptest.NewRequest("GET", "http://testing.com/foo", nil)

	// Create a ResponseRecorder, which acts as a mock ResponseWriter
	// and captures all the changes made to the response.
	rr := httptest.NewRecorder()

	// --- 2. Execution ---

	// Serve the HTTP request to our handler, which will in turn
	// use our ResponseRecorder.
	handlerToTest.ServeHTTP(rr, req)

	// --- 3. Assertions ---

	// Get the header that was set by the middleware.
	header := rr.Header().Get("Cache-Control")
	expectedHeader := fmt.Sprintf("public, max-age=%s", testMaxAge)

	// Check if the header is what we expect.
	if got, want := rr.Header().Get("Cache-Control"), fmt.Sprintf("public, max-age=%s", testMaxAge); got != want {
		t.Errorf("handler returned wrong Cache-Control header: got %q want %q", header, expectedHeader)
	}

	// Check if the status code from the "next" handler was passed through.
	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("handler returned wrong status code: got %v want %v", got, want)
	}
}
