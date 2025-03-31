package main

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/alexedwards/scs/v2"
	"github.com/sglmr/gowebstart/assets"
	"github.com/sglmr/gowebstart/internal/argon2id"
	"github.com/sglmr/gowebstart/internal/email"
	"github.com/sglmr/gowebstart/internal/render"
	"github.com/sglmr/gowebstart/internal/validator"
	"github.com/sglmr/gowebstart/internal/vcs"
)

// addRoutes adds all the routes to the mux
func addRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	devMode bool,
	mailer email.MailerInterface,
	authEmail, passwordHash string,
	wg *sync.WaitGroup,
	sessionManager *scs.SessionManager,
) {
	// Set up file server for embedded static files
	fileServer := http.FileServer(http.FS(staticFileSystem{assets.EmbeddedFiles}))
	mux.Handle("GET /static/", cacheControlMW("31536000")(fileServer))

	// Routes that don't require login or csrf
	mux.Handle("GET /", home(logger, devMode, sessionManager))
	mux.Handle("GET /health/", health(devMode))
	mux.Handle("GET /send-mail/", sendEmail(mailer, logger, wg))

	// These routes need CSRF
	dynamic := func(next http.Handler) http.Handler {
		return csrfMW(next)
	}
	mux.Handle("GET /contact/", dynamic(contact(logger, devMode, wg, mailer, sessionManager)))
	mux.Handle("POST /contact/", dynamic(contact(logger, devMode, wg, mailer, sessionManager)))
	mux.Handle("GET /login/", dynamic(login(logger, sessionManager, devMode, authEmail, passwordHash)))
	mux.Handle("POST /login/", dynamic(login(logger, sessionManager, devMode, authEmail, passwordHash)))

	// This route requires basi authentication
	basicAuthRequired := func(next http.Handler) http.Handler {
		return basicAuthMW(authEmail, passwordHash, logger)(dynamic(next))
	}
	mux.Handle("GET /basic-auth-required/", basicAuthRequired(basicAuthDemo()))

	// This route requires login
	loginRequired := func(next http.Handler) http.Handler {
		return requireLoginMW()(dynamic(next))
	}
	mux.Handle("GET /login-required/", loginRequired(loginRequiredDemo()))
	mux.Handle("GET /logout/", loginRequired(logout(logger, sessionManager, devMode)))
	mux.Handle("POST /logout/", loginRequired(logout(logger, sessionManager, devMode)))
}

//=============================================================================
//	Routes/Views/HTTP handlers
//=============================================================================

// home handles the root route
func home(
	logger *slog.Logger,
	showTrace bool,
	sessionManager *scs.SessionManager,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Redirect non-root paths to root
		// TODO: write a test for this someday
		if r.URL.Path != "/" {
			clientError(w, http.StatusNotFound)
			return
		}
		putFlashMessage(r, flashSuccess, "Welcome!", sessionManager)
		putFlashMessage(r, flashSuccess, "You made it!", sessionManager)

		data := newTemplateData(r, sessionManager)

		if err := render.Page(w, http.StatusOK, data, "home.tmpl"); err != nil {
			serverError(w, r, err, logger, showTrace)
			return
		}
	}
}

// contact handles rendering a contact page
func contact(
	logger *slog.Logger,
	showTrace bool,
	wg *sync.WaitGroup,
	mailer email.MailerInterface,
	sessionManager *scs.SessionManager,
) http.HandlerFunc {
	type contactForm struct {
		Name    string
		Email   string
		Message string
		validator.Validator
	}
	return func(w http.ResponseWriter, r *http.Request) {
		data := newTemplateData(r, sessionManager)
		data["Form"] = contactForm{}

		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				clientError(w, http.StatusBadRequest)
				return
			}

			form := contactForm{}

			// Populate the form data
			form.Name = r.FormValue("name")
			form.Email = r.FormValue("email")
			form.Message = r.FormValue("message")

			// Validate the form
			form.Check("Name", validator.NotBlank(form.Name), "Name is required.")
			form.Check("Name", validator.MaxRunes(form.Name, 100), "Name must be less than 100 characters.")

			form.Check("Email", validator.NotBlank(form.Email), "Email is required.")
			form.Check("Email", validator.IsEmail(form.Email), "Email must be a valid email address.")

			form.Check("Message", validator.NotBlank(form.Message), "Message is required.")
			form.Check("Message", validator.MaxRunes(form.Message, 1000), "Message must be less than 1,000 characters.")

			if form.Valid() {
				// Email the form message
				backgroundTask(wg, logger, func() error {
					return mailer.Send("Recipient <recipient@example.com>", "Reply-To <reply-to@example.com>", form, "example.tmpl")
				})
				// Render the contact success page
				err := render.Page(w, http.StatusFound, data, "contact-success.tmpl")
				if err != nil {
					serverError(w, r, err, logger, showTrace)
					return
				}
				return
			}

			// Update the template data form so the page errors will render
			data["Form"] = form

		}

		// Render the contact.tmpl page
		err := render.Page(w, http.StatusOK, data, "contact.tmpl")
		if err != nil {
			serverError(w, r, err, logger, showTrace)
			return
		}
	}
}

// sendEmail sends out a background email task
func sendEmail(mailer email.MailerInterface, logger *slog.Logger, wg *sync.WaitGroup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Email queued")
		emailData := map[string]any{
			"Name": "Person",
		}
		backgroundTask(
			wg, logger,
			func() error {
				return mailer.Send("Recipient <recipient@example.com>", "Reply-To <reply-to@example.com>", emailData, "example.tmpl")
			})
	}
}

// health handles a healthcheck response "OK"
func health(devMode bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "status: OK")
		fmt.Fprintln(w, "devMode:", devMode)
		fmt.Fprintln(w, "ver: ", vcs.Version())
	}
}

// basicAuthDemo handles a page protected by basic authentication.
func basicAuthDemo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "You're visiting a a page protected with basic authentication!")
	}
}

// loginRequiredDemo handles a page protected by basic authentication.
func loginRequiredDemo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "You're visiting a a page protected that required login!")
	}
}

// login handles logins
func login(
	logger *slog.Logger,
	sessionManager *scs.SessionManager,
	showTrace bool,
	authEmail, passwordHash string,
) http.HandlerFunc {
	// Login form object
	type loginForm struct {
		Email    string
		Password string
		validator.Validator
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the "next" url parameter for the page to redirect to on successful login
		nextURL := r.URL.Query().Get("next")
		logger.Debug("login next", "next", nextURL)
		if len(nextURL) == 0 {
			// Set to home if there was not next url
			nextURL = "/"
		}

		// Render form for a GET request
		if r.Method == http.MethodGet {
			data := newTemplateData(r, sessionManager)
			data["Form"] = loginForm{}

			// Render the login page
			if err := render.Page(w, http.StatusOK, data, "login.tmpl"); err != nil {
				serverError(w, r, err, logger, showTrace)
				return
			}
			return
		}

		// Parse the form data
		err := r.ParseForm()
		if err != nil {
			clientError(w, http.StatusBadRequest)
			return
		}

		// Create a form with the data
		form := loginForm{
			Email:    r.FormValue("email"),
			Password: r.FormValue("password"),
		}

		// Validate the form data
		form.Check("Email", validator.NotBlank(form.Email), "This field cannot be blank.")
		form.Check("Email", validator.MaxRunes(form.Email, 50), "This field cannot be more than 100 characters.")
		form.Check("Email", validator.IsEmail(form.Email), "Email must be a valid email.")
		form.Check("Password", validator.NotBlank(form.Password), "This field cannot be blank.")
		form.Check("Password", validator.MaxRunes(form.Password, 100), "This field cannot be more than 150 characters.")

		// Return form errors if the form is not valid
		if form.HasErrors() {
			putFlashMessage(r, flashError, "please correct the form errors", sessionManager)
			data := newTemplateData(r, sessionManager)
			data["Form"] = form

			// Render the login page
			if err := render.Page(w, http.StatusUnprocessableEntity, data, "login.tmpl"); err != nil {
				serverError(w, r, err, logger, showTrace)
				return
			}
			return
		}

		// Check if the email matches and if not, send back to the login page
		if subtle.ConstantTimeCompare([]byte(authEmail), []byte(form.Email)) == 0 {
			putFlashMessage(r, flashError, "Email or password is incorrect", sessionManager)

			data := newTemplateData(r, sessionManager)
			data["Form"] = form

			// re-render the login page
			if err := render.Page(w, http.StatusUnprocessableEntity, data, "login.tmpl"); err != nil {
				serverError(w, r, err, logger, showTrace)
				return
			}
			return
		}

		// Check whether the hashed pasword for the user and the plain text password provided match
		match, err := argon2id.ComparePasswordAndHash(form.Password, passwordHash)
		switch {
		case err != nil:
			serverError(w, r, err, logger, showTrace)
			return
		case !match:
			putFlashMessage(r, flashError, "Email or password is incorrect", sessionManager)

			data := newTemplateData(r, sessionManager)
			data["Form"] = form

			// re-render the login page
			if err := render.Page(w, http.StatusUnprocessableEntity, data, "login.tmpl"); err != nil {
				serverError(w, r, err, logger, showTrace)
				return
			}
			return
		}

		// Renew token after login to change the session ID
		err = sessionManager.RenewToken(r.Context())
		if err != nil {
			serverError(w, r, err, logger, showTrace)
			return
		}

		// Set the authenticated session key
		sessionManager.Put(r.Context(), "authenticated", true)
		putFlashMessage(r, flashSuccess, "You are in!", sessionManager)

		// Redirect to the next page.
		http.Redirect(w, r, nextURL, http.StatusSeeOther)
	}
}

// logout handles logging out
func logout(
	logger *slog.Logger,
	sessionManager *scs.SessionManager,
	showTrace bool,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the "next" url parameter for the page to redirect to on successful login
		nextURL := r.URL.Query().Get("next")
		logger.Debug("login next", "next", nextURL)
		if len(nextURL) == 0 {
			// Set to home if there was not next url
			nextURL = "/"
		}

		// Render form for a GET request
		if r.Method == http.MethodGet {
			data := newTemplateData(r, sessionManager)

			// Render the login page
			if err := render.Page(w, http.StatusOK, data, "logout.tmpl"); err != nil {
				serverError(w, r, err, logger, showTrace)
				return
			}
			return
		}

		// Renew token after login to change the session ID
		err := sessionManager.RenewToken(r.Context())
		if err != nil {
			serverError(w, r, err, logger, showTrace)
			return
		}

		// Remove the authenticated session key
		sessionManager.Remove(r.Context(), "authenticated")
		putFlashMessage(r, flashSuccess, "You've been logged out!", sessionManager)

		// Redirect to the next page.
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
