package web

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/justinas/nosurf"
)

// Middleware functions

// staticFileSystem is a custom type that embeds the standard http.FileSystem for serving static files
type staticFileSystem struct {
	fs fs.FS
}

// Open is a method on the staticFileSystem to only serve files in the
// static embedded file folder without directory listings
func (sfs staticFileSystem) Open(path string) (fs.File, error) {
	// If the file isn't in the /static directory, don't return it
	if !strings.HasPrefix(path, "static") {
		return nil, fs.ErrNotExist
	}

	// Try to open the file
	f, err := sfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	// os.Stat to determine if the path is a file or directory
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// If the file is a directory, check for an index.html file
	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := sfs.fs.Open(index); err != nil {
			closeErr := f.Close()
			if closeErr != nil {
				return nil, closeErr
			}
			return nil, err
		}
	}

	return f, nil
}

// cacheControlMW sets the Cache-Control header
func cacheControlMW(age string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%s", age))
			next.ServeHTTP(w, r)
		})
	}
}

func (app *Application) recoverPanicMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// secureHeadersMW sets security headers for the whole application
func secureHeadersMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:")
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r)
	})
}

// logRequestMW logs the http request
func logRequestMW(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var (
				ip     = r.RemoteAddr
				proto  = r.Proto
				method = r.Method
				uri    = r.URL.RequestURI()
			)
			logger.Info("request", "ip", ip, "proto", proto, "method", method, "uri", uri)
			next.ServeHTTP(w, r)
		})
	}
}

// csrfMW protects specific routes against CSRF.
func csrfMW(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	})

	return csrfHandler
}

// requireLoginMW is an HTTP middleware that enforces user authentication for protected routes.
//
// If the user is not authenticated, the middleware redirects to "/login/". The original request URI is appended to the "/login/" with a 'next' query parameter (ex. "/login/?next=/account/").
func requireLoginMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect to login if the user isn't authenticated
		if !isAuthenticated(r) {
			redirectURL := "/login/?next=" + url.QueryEscape(r.RequestURI)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}

		// Set cache control to no-store so that these pages aren't cached
		w.Header().Add("Cache-Control", "no-store")

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// authenticateMW is an HTTP middleware that checks for an authenticated user session
// and makes the authentication status available in the request context. Storing session details
// in the context about authentication reduces I/O operations against the database when checking
// for authentication and common user data in tempates and routes/handlers.
//
// authenticateMW retrieves the "authenticated" status from the app.SessionManager.
// If the user is authenticated, it creates a new request context with
// `isAuthenticatedContextKey` set to `true` and updates the request.
// This allows subsequent handlers in the chain to easily determine if the user is logged in.
// This middleware does not restrict access. Access is restricted by the requireLoginMW().
func (app *Application) authenticateMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the authenticated user ID from the session manager. This
		// key is set on the login() handler when a user successfully logs in.
		// If the user did not log in, serve the next http request.
		id := app.SessionManager.GetInt64(r.Context(), "authenticatedUserID")
		if id == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Check that user exists in the database
		user, err := app.Queries.GetUserByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// TODO: This should probably be some other thing, like a Unauthorized error.
				next.ServeHTTP(w, r)
				return
			}
			app.serverError(w, r, err)
		}

		// If the user exists then create a new copy of the request with
		//	- userContextKey = db.User object
		// 	- isAuthenticatedContextKey = true
		r = app.contextSetUser(r, user)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// trailingSlashMiddleware redirects paths without a trailing slash to their trailing-slash equivalent.
//
// A redirect is not issued when:
//  1. Path is root "/"
//  2. Path already has a trailing slash
//  3. Path starts with "/static/"
func trailingSlashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the current request path.
		path := r.URL.Path

		// Serve the next http.Handler if no redirect is needed.
		if !redirectWithSlash(path) {
			next.ServeHTTP(w, r)
			return
		}

		// Rebuild the URL with the trailing slash and the original query string (if any).
		newURL := path + "/"
		if r.URL.RawQuery != "" {
			newURL += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, newURL, http.StatusMovedPermanently)
	})
}

// redirectWithSlash returns true if a URL should be redirected
// with a '/' suffix added and false otherwise. This function expects
// a url.URL.Path which does not include query parameters.
//
// Redirects are skipped for paths that already end in "/"
// and for anything served from "/static/..."
func redirectWithSlash(path string) bool {
	// Check for the conditions where we should NOT redirect.
	switch {
	// Path is root "/" or already has a trailing slash.
	case strings.HasSuffix(path, "/"):
		return false
	// Path is "/static/"
	case strings.HasPrefix(path, "/static/"):
		return false
	}

	return true
}
