package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/sglmr/gowebstart/internal/assert"
	"github.com/sglmr/gowebstart/internal/vcs"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	defer ts.Close()

	response := ts.get(t, "/health/")

	// Check that the status code was 200.
	assert.Equal(t, http.StatusOK, response.statusCode)

	// Check the content type
	assert.Equal(t, response.header.Get("Content-Type"), "text/plain")

	// Check the body contains "OK"
	assert.StringIn(t, "status: OK", response.body)
	assert.StringIn(t, vcs.Version(), response.body)
}

func TestContactE2E(t *testing.T) {
	t.Parallel()

	// Create a new test server
	ts := newTestServer(t)
	defer ts.Close()

	// ------- Test GET Method ---------

	response := ts.get(t, "/contact/")
	token := response.csrfToken(t)

	// Check the status of the request
	assert.Equal(t, response.statusCode, http.StatusOK)

	// Check that the body contains the word "contact"
	assert.StringIn(t, "Contact", response.body)

	// -------- Test Post without CSRF --------------------

	data := url.Values{}
	data.Add("name", "joe")
	data.Add("email", "joe@example.com")
	data.Add("message", "some message")

	// Create a new http POST request.
	response = ts.post(t, "/contact/", data)

	// Bad request because
	assert.Equal(t, response.statusCode, http.StatusBadRequest)

	// --------- Test POST with CSRF -----------------

	// Add the csrf_token to the request
	data.Add("csrf_token", token)

	// Create a new http POST request.
	response = ts.post(t, "/contact/", data)

	assert.Equal(t, response.statusCode, http.StatusFound)
}

func TestHome(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t)
	defer ts.Close()

	response := ts.get(t, "/")

	assert.Equal(t, http.StatusOK, response.statusCode)
	assert.StringIn(t, "Example", response.body)
}

func TestLoginLogout(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// Test logout unauthorized without login
	response := ts.get(t, "/logout/")
	assert.Equal(t, http.StatusSeeOther, response.statusCode)

	// Test login without login
	response = ts.get(t, "/login/")
	assert.Equal(t, http.StatusOK, response.statusCode)
	assert.StringIn(t, `<input type="hidden" name="csrf_token"`, response.body)
	assert.StringIn(t, `<input type="text" id="email" name="email"`, response.body)
	assert.StringIn(t, `<input type="password" id="password" name="password"`, response.body)
	assert.StringNotIn(t, `/logout/`, response.body)

	// Try login with fake username
	data := url.Values{}
	data.Set("csrf_token", response.csrfToken(t))
	data.Set("email", "fake@example.com")
	data.Set("password", testPassword)
	response = ts.post(t, "/login/", data)
	assert.Equal(t, http.StatusUnprocessableEntity, response.statusCode)
	assert.StringIn(t, "Email or password is incorrect", response.body)
	assert.StringNotIn(t, "You are in!", response.body)

	// Try login with a fake password
	data.Set("email", testEmail)
	data.Set("password", "wrong-password")
	response = ts.post(t, "/login/", data)
	assert.Equal(t, http.StatusUnprocessableEntity, response.statusCode)
	assert.StringIn(t, "Email or password is incorrect", response.body)
	assert.StringNotIn(t, "You are in!", response.body)

	// Try login with real password and email
	data.Set("email", testEmail)
	data.Set("password", testPassword)
	response = ts.post(t, "/login/", data)
	assert.Equal(t, http.StatusSeeOther, response.statusCode)

	// Check flash message on next page
	response = ts.get(t, "/")
	assert.StringIn(t, "You are in!", response.body)
	assert.StringNotIn(t, "Email or password is incorrect", response.body)

	// Try logout get after login
	response = ts.get(t, "/logout/")
	assert.Equal(t, http.StatusOK, response.statusCode)

	// Try posting logout to log out
	data = url.Values{}
	data.Set("csrf_token", response.csrfToken(t))
	response = ts.post(t, "/logout/", data)
	assert.Equal(t, http.StatusSeeOther, response.statusCode)

	// Logout get should redirect to login page now
	response = ts.get(t, "/logout/")
	assert.Equal(t, http.StatusSeeOther, response.statusCode)
}
