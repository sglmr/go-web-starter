package web

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/justinas/nosurf"
)

//=============================================================================
// Middleware functions
//=============================================================================

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
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					// we don't recover http.ErrAbortHandler so the response
					// to the client is aborted, this should not be logged
					panic(rvr)
				}
				app.serverError(w, r, fmt.Errorf("%s", rvr))
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
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

// requireLoginMW checks if a user is authenticated, and if not, redirects them to the login page.
func requireLoginMW() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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
}

// authenticateMW sets a context isAuthenticatedContextKey to true if a user is authenticated
// This middleware can also add user attributes to the request context to reduce queries for user or session data to the database.
func authenticateMW(SessionManager *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authenticated := SessionManager.GetBool(r.Context(), "authenticated")
			if !authenticated {
				next.ServeHTTP(w, r)
				return
			}

			// Check that user exists in the database
			// TODO with database: Not applicable without a users table

			// If the user exists then create a new copy of the request
			// with the isAuthenticatedContextKey set to true
			ctx := context.WithValue(r.Context(), isAuthenticatedContextKey, true)
			ctx = context.WithValue(ctx, isAnonyousContextKey, true)
			r = r.WithContext(ctx)

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// trailingSlashMiddleware redirects paths without a trailing slash to their trailing-slash equivalent.
// it does not redirect to a trailing slash when:
//  1. If the path is the root "/"
//  2. If the path already has a trailing slash
//  3. If the path starts with "/static/"
func trailingSlashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the current request path.
		path := r.URL.Path

		// Check for the conditions where we should NOT redirect.
		// 1. If the path is the root "/".
		// 2. If the path already has a trailing slash.
		// 3. If the path starts with "/static/".
		if path != "/" && !strings.HasSuffix(path, "/") && !strings.HasPrefix(path, "/static/") {
			// Construct the new URL with a trailing slash.
			newPath := path + "/"

			// Perform a 301 Permanent Redirect.
			http.Redirect(w, r, newPath, http.StatusMovedPermanently)
			return // Stop further processing.
		}

		// If no redirect is needed, pass the request to the next handler.
		next.ServeHTTP(w, r)
	})
}
