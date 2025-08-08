package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sglmr/gowebstart/internal/db"
	"github.com/sglmr/gowebstart/internal/render"
	"github.com/sglmr/gowebstart/internal/validator"
	"github.com/sglmr/gowebstart/internal/vcs"
)

//=============================================================================
//	Routes/Views/HTTP handlers
//=============================================================================

// home handles the root route
func (app *Application) home(w http.ResponseWriter, r *http.Request) {
	// Redirect non-root paths to root
	// TODO: write a test for this someday
	if r.URL.Path != "/" {
		clientError(w, http.StatusNotFound)
		return
	}
	app.Flash(r, flashSuccess, "Welcome!")
	app.Flash(r, flashSuccess, "You made it!")

	data := newTemplateData(r, app.SessionManager)

	if err := render.Page(w, http.StatusOK, data, "home.tmpl"); err != nil {
		app.serverError(w, r, err)
		return
	}
}

// contact handles rendering a contact page
func (app *Application) contact() http.HandlerFunc {
	type contactForm struct {
		Name    string
		Email   string
		Message string
		validator.Validator
	}
	return func(w http.ResponseWriter, r *http.Request) {
		data := newTemplateData(r, app.SessionManager)
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
				app.Background(func() error {
					return app.Email.Send("Recipient <recipient@example.com>", "Reply-To <reply-to@example.com>", form, "example.tmpl")
				})
				// Render the contact success page
				err := render.Page(w, http.StatusSeeOther, data, "contact-success.tmpl")
				if err != nil {
					app.serverError(w, r, err)
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
			app.serverError(w, r, err)
			return
		}
	}
}

// sendEmail sends out a background email task
func (app *Application) sendEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "Email queued")
	emailData := map[string]any{
		"Name": "Person",
	}
	app.Background(

		func() error {
			return app.Email.Send("Recipient <recipient@example.com>", "Reply-To <reply-to@example.com>", emailData, "example.tmpl")
		})
}

// health handles a healthcheck response "OK"
func (app *Application) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "status: OK")
	fmt.Fprintln(w, "devMode:", app.DevMode)
	fmt.Fprintln(w, "ver: ", vcs.Version())
}

// loginRequiredDemo handles a page protected by basic authentication.
func (app *Application) loginRequiredDemo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "You're visiting a a page protected that required login!")
	}
}

// login handles logins
func (app *Application) login() http.HandlerFunc {
	// Login form object
	type loginForm struct {
		Email    string
		Password string
		validator.Validator
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// GET request response.
		// Render the login page with an empty form.
		if r.Method == http.MethodGet {
			data := newTemplateData(r, app.SessionManager)
			data["Form"] = loginForm{}

			// Render the login page
			if err := render.Page(w, http.StatusOK, data, "login.tmpl"); err != nil {
				app.serverError(w, r, err)
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
			app.Flash(r, flashError, "please correct the form errors")
			data := newTemplateData(r, app.SessionManager)
			data["Form"] = form

			// Render the login page
			if err := render.Page(w, http.StatusUnprocessableEntity, data, "login.tmpl"); err != nil {
				app.serverError(w, r, err)
				return
			}
			return
		}

		// Authenticate the user
		_, err = app.Queries.AuthenticateLogin(r.Context(), form.Email, form.Password)
		if err != nil {
			// Reload the page with failed login message
			if errors.Is(err, db.ErrAuthFailed) {
				app.Flash(r, flashError, "login failed")

				data := newTemplateData(r, app.SessionManager)
				data["Form"] = form
				// re-render the login page with the form data
				if err := render.Page(w, http.StatusUnauthorized, data, "login.tmpl"); err != nil {
					app.serverError(w, r, err)
					return
				}
				return
			}
			// Any other error that isn't ErrAuthFailed is a server error
			app.serverError(w, r, err)
			return
		}

		// Renew token after login to change the session ID
		err = app.SessionManager.RenewToken(r.Context())
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		// Set the authenticated session key
		app.SessionManager.Put(r.Context(), "authenticated", true)
		app.Flash(r, flashSuccess, "You are in!")

		// Get the 'next=' query parameter for the next page
		// to redirect the user to.
		nextURL := r.URL.Query().Get("next")
		app.Log.Debug("login next", "next", nextURL)
		// Rederect to the homepage if there was no 'next=' query parameter.
		if len(nextURL) == 0 {
			nextURL = "/"
		}

		// Redirect to the next page.
		http.Redirect(w, r, nextURL, http.StatusSeeOther)
	}
}

// logout handles logging out
func (app *Application) logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Render form for a GET request
		if r.Method == http.MethodGet {
			data := newTemplateData(r, app.SessionManager)

			// Render the login page
			if err := render.Page(w, http.StatusOK, data, "logout.tmpl"); err != nil {
				app.serverError(w, r, err)
				return
			}
			return
		}

		// Renew token after login to change the session ID
		err := app.SessionManager.RenewToken(r.Context())
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		// Remove the authenticated session key
		app.SessionManager.Remove(r.Context(), "authenticated")
		app.Flash(r, flashSuccess, "You've been logged out!")

		// Redirect to the next page.
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
