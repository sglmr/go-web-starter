package web

import (
	"encoding/gob"
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
	"github.com/sglmr/gowebstart/internal/data"
	"github.com/sglmr/gowebstart/internal/email"
)

func init() {
	gob.Register(FlashMessage{})
	gob.Register([]FlashMessage{})
}

type Application struct {
	Log            *slog.Logger
	DevMode        bool
	Email          email.MailerInterface
	SessionManager *scs.SessionManager
	Queries        *data.Queries
	wg             *sync.WaitGroup
}

// NewApplication creates and returns a new Application struct. It initializes
// all the necessary components, such as the logger, email service, session
// manager, and database queries. This function is the central point for
// setting up the application's dependencies.
func NewApplication(logger *slog.Logger,
	devMode bool,
	mailer email.MailerInterface,
	sessionManager *scs.SessionManager,
	queries *data.Queries,
) *Application {
	return &Application{
		Log:            logger,
		DevMode:        devMode,
		Email:          mailer,
		SessionManager: sessionManager,
		Queries:        queries,
		wg:             &sync.WaitGroup{},
	}
}

// NewHandler creates and returns a new http.Handler with all the middleware
// and routes configured.
func (app *Application) NewHandler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(logRequestMW(app.Log))
	r.Use(app.recoverPanicMW)
	r.Use(trailingSlashMiddleware)
	r.Use(secureHeadersMW)
	r.Use(app.SessionManager.LoadAndSave)
	r.Use(app.authenticateMW)

	// Set at imeout value on the request context
	r.Use(middleware.Timeout(60 * time.Second))

	// Set up file server for embedded static files
	fileServer := http.FileServer(http.FS(staticFileSystem{assets.EmbeddedFiles}))
	r.Group(func(r chi.Router) {
		r.Use(cacheControlMW("31536000"))
		r.Mount("/static", fileServer)
	})

	r.Get("/", app.home)
	r.Get("/health/", app.health)
	r.Get("/send-mail/", app.sendEmail)

	// These routes need CSRF
	r.Group(func(r chi.Router) {
		r.Use(csrfMW)
		r.Get("/contact/", app.contact())
		r.Post("/contact/", app.contact())
		r.Get("/login/", app.login())
		r.Post("/login/", app.login())
	})

	// This route requires login
	r.Group(func(r chi.Router) {
		r.Use(csrfMW)
		r.Use(requireLoginMW)
		r.Get("/login-required/", app.loginRequiredDemo())
		r.Get("/logout/", app.logout())
		r.Post("/logout/", app.logout())
	})

	return r
}

// Background runs a function in a separate goroutine. It's designed for tasks
// that can be executed asynchronously, such as sending emails or processing
// data, without blocking the main application flow. It includes error handling
// and panic recovery to ensure that background tasks don't crash the server.
func (app *Application) Background(task func() error) {
	// Increment waitgroup to track whether this background task is complete or not
	app.wg.Add(1)

	// Launch a goroutine to run the task in
	go func() {
		// decrement the waitgroup after the task completes
		defer app.wg.Done()

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
