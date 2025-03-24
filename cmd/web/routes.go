package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/alexedwards/scs/v2"
	"github.com/sglmr/gowebstart/assets"
	"github.com/sglmr/gowebstart/internal/email"
	"github.com/sglmr/gowebstart/internal/render"
	"github.com/sglmr/gowebstart/internal/validator"
	"github.com/sglmr/gowebstart/internal/vcs"
)

// AddRoutes adds all the routes to the mux
func AddRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	devMode bool,
	mailer email.MailerInterface,
	username, password string,
	wg *sync.WaitGroup,
	sessionManager *scs.SessionManager,
) http.Handler {
	// Set up file server for embedded static files
	// fileserver := http.FileServer(http.FS(assets.EmbeddedFiles))
	fileServer := http.FileServer(http.FS(staticFileSystem{assets.EmbeddedFiles}))
	mux.Handle("GET /static/", CacheControlMW("31536000")(fileServer))

	mux.Handle("GET /", home(logger, devMode, sessionManager))
	// TODO: Figure out how to wrap this with nosurf
	c := contact(logger, devMode, wg, mailer, sessionManager)
	mux.Handle("GET /contact/", CsrfMW(c))
	mux.Handle("POST /contact/", CsrfMW(c))
	mux.Handle("GET /health/", health())
	mux.Handle("GET /send-mail", sendEmail(mailer, logger, wg))

	mux.Handle("GET /protected/", BasicAuthMW(username, password, logger)(protected()))

	// Add recoverPanic middleware
	handler := RecoverPanicMW(mux, logger, devMode)
	handler = SecureHeadersMW(handler)
	handler = LogRequestMW(logger)(handler)
	handler = sessionManager.LoadAndSave(handler)

	// Return the handler
	return handler
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
			NotFound(w, r)
			return
		}
		putFlashMessage(r, LevelSuccess, "Welcome!", sessionManager)
		putFlashMessage(r, LevelSuccess, "You made it!", sessionManager)

		data := newTemplateData(r, sessionManager)

		if err := render.Page(w, http.StatusOK, data, "home.tmpl"); err != nil {
			ServerError(w, r, err, logger, showTrace)
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
				BadRequest(w, r, err)
				return
			}

			form := contactForm{}

			// Populate the form data
			form.Name = r.FormValue("name")
			form.Email = r.FormValue("email")
			form.Message = r.FormValue("message")

			// Validate the form
			form.Check(validator.NotBlank(form.Name), "Name", "Name is required.")
			form.Check(validator.MaxRunes(form.Name, 100), "Name", "Name must be less than 100 characters.")

			form.Check(validator.NotBlank(form.Email), "Email", "Email is required.")
			form.Check(validator.IsEmail(form.Email), "Email", "Email must be a valid email address.")

			form.Check(validator.NotBlank(form.Message), "Message", "Message is required.")
			form.Check(validator.MaxRunes(form.Message, 1000), "Message", "Message must be less than 1,000 characters.")

			if form.Valid() {
				// Email the form message
				BackgroundTask(wg, logger, func() error {
					return mailer.Send("Recipient <recipient@example.com>", "Reply-To <reply-to@example.com>", form, "example.tmpl")
				})
				// Render the contact success page
				err := render.Page(w, http.StatusFound, data, "contact-success.tmpl")
				if err != nil {
					ServerError(w, r, err, logger, showTrace)
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
			ServerError(w, r, err, logger, showTrace)
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
		BackgroundTask(
			wg, logger,
			func() error {
				return mailer.Send("Recipient <recipient@example.com>", "Reply-To <reply-to@example.com>", emailData, "example.tmpl")
			})
	}
}

// health handles a healthcheck response "OK"
func health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "status: OK")
		fmt.Fprintln(w, "ver: ", vcs.Version())
	}
}

// protected handles a page protected by basic authentication.
func protected() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "You're visiting a protected page!")
	}
}
