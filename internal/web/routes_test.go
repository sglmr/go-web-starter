package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sglmr/gowebstart/internal/assert"
	"github.com/sglmr/gowebstart/internal/vcs"
)

func TestHome(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	response := ts.get(t, "/")

	// Check statusCode == 200
	if got, want := response.statusCode, http.StatusOK; got != want {
		t.Fatalf("/ statusCode %v, want %v", got, want)
	}

	// Check "Example" is in the response body
	if want := "Example"; !strings.Contains(response.body, want) {
		t.Errorf("/ body missing %q", want)
	}

	// Check body contains flash message
	if got, want := response.body, "You made it!"; !strings.Contains(got, want) {
		t.Errorf("/ body missing %q", want)
	}
}

func TestHealth(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	response := ts.get(t, "/health/")

	// Check statusCode == 200
	if got, want := response.statusCode, http.StatusOK; want != got {
		t.Errorf("/health/ statusCode %v, want %v", got, want)
	}

	// Check Content-Type header
	if got, want := response.header.Get("Content-Type"), "text/plain"; got != want {
		t.Errorf("/health/ Content-Type %q, want %q", got, want)
	}

	// Check the body contains "status: OK"
	if got, want := response.body, "status: OK"; !strings.Contains(got, want) {
		t.Errorf("/health/ body missing %q", want)
	}

	// Check the body contains the vcs.Version()
	if got, want := response.body, vcs.Version(); !strings.Contains(got, want) {
		t.Errorf("/health/ body missing version, %q", want)
	}
}

func TestContactGet(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	response := ts.get(t, "/contact/")

	// Check statusCode == 200
	if got, want := response.statusCode, http.StatusOK; got != want {
		t.Errorf("/contact/ statusCode %v, want %v", got, want)
	}

	// Check the body contents
	if got, want := response.body, "Contact"; !strings.Contains(got, want) {
		t.Errorf("/contact/ body missing %q", want)
	}

	// Check a CSRF token exists
	if got, want := response.csrfToken(t), 0; len(got) <= want {
		t.Errorf("/contact/ missing CSRF token")
	}

	// TODO: build more tests to check the page contains the expected form fields.
}

func TestContactPost(t *testing.T) {
	// Create a new test server
	ts := newTestServer(t)
	defer ts.Close()

	// Get a CSRF token
	response := ts.get(t, "/contact/")
	token := response.csrfToken(t)

	// Create data without CSRF token
	data := url.Values{}
	data.Add("name", "joe")
	data.Add("email", "joe@example.com")
	data.Add("message", "some message")

	// Create a new http POST request.
	response = ts.post(t, "/contact/", data)

	// Should be a bad request
	if got, want := response.statusCode, http.StatusBadRequest; got != want {
		t.Errorf("/contact/ status without CSRF was %v, want %v", got, want)
	}

	// Make a request with the CSRF token
	data.Add("csrf_token", token)
	response = ts.post(t, "/contact/", data)

	// Check for status after POST with CSRF token
	if got, want := response.statusCode, http.StatusSeeOther; got != want {
		t.Errorf("/contact/ status with CSRF was %v, want %v", got, want)
	}

	// TODO: run through validations and body/page contents with missing data fields.
}

func TestLoginLogout(t *testing.T) {
	successMessage := "You are in!"

	ts := newTestServer(t)
	defer ts.Close()

	// Test logout unauthorized without login
	response := ts.get(t, "/logout/")
	assert.Equal(t, http.StatusSeeOther, response.statusCode)

	// Test login page contents
	response = ts.get(t, "/login/")

	if got, want := response.statusCode, http.StatusOK; got != want {
		t.Fatalf("/login/ statusCode was %v, want %v", got, want)
	}

	// Table driven tests for login page contents
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "csrf token field",
			expected: `<input type="hidden" name="csrf_token"`,
		},
		{
			name:     "email field",
			expected: `<input type="text" id="email" name="email"`,
		},
		{
			name:     "password field",
			expected: `<input type="password" id="password" name="password"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := response.body, tt.expected; !strings.Contains(got, want) {
				t.Errorf("/login/ body missing %v", want)
			}
		})
	}

	// Logout should not display on the login page
	if got, want := response.body, "/logout"; strings.Contains(got, want) {
		t.Errorf("/login/ page should not have %q", want)
	}

	// Try login with fake username
	data := url.Values{}
	data.Set("csrf_token", response.csrfToken(t))
	data.Set("email", "fake@example.com")
	data.Set("password", testPassword)
	response = ts.post(t, "/login/", data)

	// Fake username should not work
	if got, want := response.statusCode, http.StatusUnprocessableEntity; got != want {
		t.Fatalf("/login/ fake username statusCode was %v, want %v", got, want)
	}
	// There should be an error flash message
	if got, want := response.body, "login failed"; !strings.Contains(got, want) {
		t.Errorf("/login/ fake username body is missing %q", want)
	}
	// There should not be a success flash message
	if got, want := response.body, successMessage; strings.Contains(got, want) {
		t.Errorf("/login/ fake username body shouldn't have %q", want)
	}

	// Try login with a fake password
	data.Set("email", testEmail)
	data.Set("password", "wrong-password")
	response = ts.post(t, "/login/", data)

	// Fake password should not work
	if got, want := response.statusCode, http.StatusUnprocessableEntity; got != want {
		t.Fatalf("/login/ fake password statusCode was %v, want %v", got, want)
	}
	// There should be an error flash message
	if got, want := response.body, "login failed"; !strings.Contains(got, want) {
		t.Errorf("/login/ fake password body is missing %q", want)
	}
	// There should not be a success flash message
	if got, want := response.body, successMessage; strings.Contains(got, want) {
		t.Errorf("/login/ fake password body shouldn't have %q", want)
	}

	// Try login with real password and email
	data.Set("admin@example.com", testEmail)
	data.Set("secret", testPassword)
	response = ts.post(t, "/login/", data)
	if got, want := response.statusCode, http.StatusSeeOther; got != want {
		t.Fatalf("/login/ real creds statusCode was %v, want %v", got, want)
	}

	// Check for success flash message on next page
	response = ts.get(t, "/")
	if got, want := response.body, successMessage; !strings.Contains(got, want) {
		t.Errorf("/login/ real creds did not contain %q", want)
	}

	// Check fail success message wasn't on the page.
	if got, want := response.body, "Email or password is incorrect"; strings.Contains(got, want) {
		t.Errorf("/login/ real creds should not contain %q", want)
	}

	// Try logout get after login
	response = ts.get(t, "/logout/")

	// Try getting the logout page, it should work now
	if got, want := response.statusCode, http.StatusOK; got != want {
		t.Fatalf("/logout/ GET statusCode was %v, want %v", got, want)
	}
	assert.Equal(t, http.StatusOK, response.statusCode)

	// Try posting logout to log out
	data = url.Values{}
	data.Set("csrf_token", response.csrfToken(t))
	response = ts.post(t, "/logout/", data)

	// Logout POST should succeed
	if got, want := response.statusCode, http.StatusSeeOther; got != want {
		t.Fatalf("/logout/ POST statusCode was %v, want %v", got, want)
	}

	// Logout get should redirect to login page now
	response = ts.get(t, "/logout/")
	if got, want := response.statusCode, http.StatusSeeOther; got != want {
		t.Fatalf("/logout/ GET statusCode was %v, want %v", got, want)
	}
}
