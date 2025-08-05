package web

import (
	"log/slog"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sglmr/gowebstart/assets"
	"github.com/sglmr/gowebstart/internal/db"
	"github.com/sglmr/gowebstart/internal/email"
)

type Application struct {
	Log                              *slog.Logger
	DevMode                          bool
	Email                            email.MailerInterface
	AdminUsername, AdminPasswordHash string
	Wg                               *sync.WaitGroup
	SessionManager                   *scs.SessionManager
	DB                               *db.DB
}

// NewHandler creates a new htp.Handler with all the middlware and routes configured.
func (app *Application) NewHandler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(app.recoverPanicMW)
	r.Use(secureHeadersMW)
	r.Use(middleware.StripSlashes)
	r.Use(app.SessionManager.LoadAndSave)
	r.Use(authenticateMW(app.SessionManager))
	r.Use(logRequestMW(app.Log))

	// Set at imeout value on the request context
	r.Use(middleware.Timeout(60 * time.Second))

	// Set up file server for embedded static files
	fileServer := http.FileServer(http.FS(staticFileSystem{assets.EmbeddedFiles}))
	r.With(cacheControlMW("31536000")).Mount("/static/", fileServer)

	r.Get("/", app.home)
	r.Get("/health", app.health)
	r.Get("/send-mail", app.sendEmail)

	// These routes need CSRF
	r.Group(func(r chi.Router) {
		r.Use(csrfMW)
		r.Get("/contact", app.contact())
		r.Post("/contact", app.contact())
		r.Get("/login", app.login())
		r.Post("/login", app.login())
	})

	// These routes need basic authentication
	r.Group(func(r chi.Router) {
		r.Use(csrfMW)
		r.Use(basicAuthMW(app.AdminUsername, app.AdminPasswordHash, app.Log))
		r.Get("/basic-auth-required", app.basicAuthDemo())
	})

	// This route requires login
	r.Group(func(r chi.Router) {
		r.Use(csrfMW)
		r.Use(requireLoginMW())
		r.Get("/login-required", app.basicAuthDemo())
		r.Get("/logout", app.logout())
		r.Post("/logout", app.logout())
	})

	return r
}

// Background executes a function in a background goroutine with proper error handling.
func (app *Application) Background(task func() error) {
	// Increment waitgroup to track whether this background task is complete or not
	app.Wg.Add(1)

	// Launch a goroutine to run the task in
	go func() {
		// decrement the waitgroup after the task completes
		defer app.Wg.Done()

		// Get the name of the function
		funcName := runtime.FuncForPC(reflect.ValueOf(task).Pointer()).Name()

		// Recover any panics in the task function so that the server doesn't die.
		defer func() {
			err := recover()
			if err != nil {
				app.Log.Error("background task recovered from panic", "task", funcName, "error", err)
			}
		}()

		// Execute the provided function, logging any errors.
		err := task()
		if err != nil {
			app.Log.Error("background task error", "task", funcName, "error", err)
		}
	}()
}
