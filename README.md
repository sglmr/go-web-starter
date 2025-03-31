# Go Start

A lightweight, feature-rich Go web application template with built-in security features, session management, email functionality, authentication, and more.

This project template aims to reduce third party dependencies wherever possible. Most of the "code" for the project is in a single `cmd/web/main.go` file. Because (A) it's easier to give AI project context when most of the project is in a single file and (B) it's a simple starter template with minimal assumptions about how a project might evolve over time.

This project does not currently include any configuration for a database.

This project aims to avoid using receiver methods on handlers and other project functions. An application struct hanging off of every method is convenient and makes for pretty code, but there are also some negatives:

- Surprise dependency issues during testing
- Non-implicit dependencies
- More restrictive coupling of project components and structure

This project assumes you will be running it behind a reverse proxy service that handles HTTPS and certificates for you.

## Features

- **Complete Web Server**: HTTP server with graceful shutdown
- **Authentication System**: Login/Logout functionality with session management
- **Middleware Stack**:
  - Panic recovery
  - Secure headers
  - Request logging
  - CSRF protection
  - Basic authentication
  - Static asset caching
  - Session management
- **Email Support**: Send emails with configurable SMTP
- **Form Validation**: Comprehensive validation helpers
- **Flash Messages**: Session-based notifications system
- **Templating**: HTML template rendering with data context
- **TailwindCSS**: Style HTML pages with TailwindCSS
- **Static File Serving**: Embedded static file handling
- **Development Mode**: Enhanced debugging with stack traces and additional logging
- **Live Reload**: Live reload with [air](https://github.com/air-verse/air)

## Getting Started

### Prerequisites

- Go 1.22 or higher
- [Task](https://taskfile.dev/) for project management commands
- [Air](https://github.com/air-verse/air) for live reload
- [Tailwind CSS](https://tailwindcss.com) for CSS

### Tailwind Installation

Follow the CLI instructions: https://tailwindcss.com/docs/installation/tailwind-cli

```sh
# Install Tailwind & plugins
npm install tailwindcss @tailwindcss/cli
npm install -D @tailwindcss/forms
npm install -D @tailwindcss/typography
```

### Installation

1. Clone the repository:

```bash
git clone https://github.com/sglmr/gowebstart.git
cd gowebstart
```

2. Run `npm install` to download tailwind dependencies.

3. Try `task run:live` to make sure the project runs.

4. Replace "gowebstart" with your new project name.

### Running the Server

Basic usage:

```bash
# Run the app
task run

# Run the app with live reload (includes rebuilding tailwind css)
task run:live
```

This will start the server on the default address `0.0.0.0:8000`.

### Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-host` | Server host | `0.0.0.0` |
| `-port` | Server port | `8000` or `PORT` env variable |
| `-dev` | Development mode | `false` |
| `-auth-email` | Basic auth admin email | `admin` |
| `-auth-password-hash` | Basic auth admin password hash | `password` (hashed) |
| `-smtp-host` | SMTP server host | `` |
| `-smtp-port` | SMTP server port | `25` |
| `-smtp-username` | SMTP username | `` |
| `-smtp-password` | SMTP password | `` |
| `-smtp-from` | Email sender | `Example Name <no-reply@example.com>` |
| `-send-email` | Send live emails | `false` |

Example with custom options:

```bash
./gowebstart -port=3000 -dev -smtp-host=smtp.example.com -smtp-port=587 -smtp-username=user -smtp-password=pass
```

## Authentication

The template includes basic authentication and login/logout functionality.

### Basic Authentication

The project contains a `BasicAuthMW` middleware that you can use to protect the application or specific routes with HTTP basic authentication.

You can try this out by visiting the [http://localhost:8000/basic-auth-required/](http://localhost:8000/basic-auth-required/) endpoint in any web browser and entering the default email and password.

### Login/Logout System

The application also includes a more user-friendly login and logout system through the web interface.

- Login page: [http://localhost:8000/login/](http://localhost:8000/login/)
- Logout page: [http://localhost:8000/logout/](http://localhost:8000/logout/)

Protected routes can be set up using the `requireLoginMW` middleware.

### Creating Password Hashes

You can use the included `hash` tool to generate secure password hashes:

```sh
go run ./cmd/hash

   Enter password: 
Re-enter password:
    Password hash: $2a$10$xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## SMTP Emails

The application includes methods for sending SMTP Emails. Email templates are configurable in the `assets/emails` directory.

```go
err = mailer.Send(recipient string, replyTo string, data any, templates ...string)
```

## Background Tasks

The application includes a system for running asynchronous tasks using the `backgroundTask` function.

```go
backgroundTask(wg *sync.WaitGroup, logger *slog.Logger, fn func() error)
```

Background task system features:

- **Panic Recovery**: Tasks are isolated so panics don't crash the server
- **Logging**: Automatic error logging with the function name
- **WaitGroup Integration**: Proper shutdown handling with sync.WaitGroup
- **Graceful Shutdown**: Tasks tracked during server shutdown

Example usage:

```go
// Send an email in the background
backgroundTask(
    wg, logger, 
    func() error {
        return mailer.Send("recipient@example.com", "reply-to@example.com", emailData, "email-template.tmpl")
    })

// Continue processing the request without waiting
```

This pattern is useful for operations like:

- Sending emails
- Processing uploaded files
- Running reports
- Performing database maintenance
- Any long-running task that shouldn't block the request handler

## Architecture

### Application Structure

- `assets/`: Folder for all project embedded files
  - `emails/`: Email templates
  - `migrations/`: Database migration files
  - `static/`: Static files like CSS, JavaScript, etc.
  - `templates/`: Templates to render to HTML pages for the application
    - `pages/`: Main web page content to load, like "home.tmpl" or "about.tmpl"
    - `partials/`: Page partials, like a nav bar, footer, etc.
    - `base.tmpl`: Base template for all pages and partials
  - `efs.go`: Specify assets folders to include in the Go binary build
  - `tailwind.css`: Input file for Tailwind CSS
- `cmd/`
  - `hash/`
    - `hash.go`: CLI tool for hashing passwords with argon2id
  - `web/`
    - `helpers.go`: Template, response, and flash message helpers for the application
    - `middleware.go`: Middleware used by the application
    - `routes.go`: Route configuration & handlers for the application
    - `main.go`: Entry point and server configuration
- `internal/`:
  - `argon2id/`: Vendored in package of [github.com/alexedwards/argon2id](https://github.com/alexedwards/argon2id)
  - `assert/`: Testing assert functions
  - `email/`: SMTP email functionality
  - `funcs/`: Template functions
  - `render/`: Template rendering helpers
  - `validator/`: Form validation
  - `vcs/`: Version information
- `.air.toml`: Live reload configuration
- `Taskfile.yml`: Project tasks ran with `task` prefix.

### Middleware

The application uses a composable middleware pattern:

```go
handler = recoverPanicMW(mux, logger, devMode)
handler = secureHeadersMW(handler)
handler = logRequestMW(logger)(handler)
handler = sessionManager.LoadAndSave(handler)
```

## Customization

### Adding New Routes and Middleware

Add new routes and middleware in the `addRoutes` function. This project takes advantage of the [Go 1.22 Routing Enhancements](https://go.dev/blog/routing-enhancements).

```go
func addRoutes(mux *http.ServeMux, ...) {
    // Existing routes...
    
    // Add your new route
    mux.Handle("GET /your-path", yourHandler(dependencies...))
    
    // Middleware...
    handler := middleware1(mux)
    handler = middleware2(handler)

    return handler
}
```

### Rendering pages from templates

Templates are rendered using the `render.Page` function. Template pages live in the `assets/templates/pages` directory.

A `newTemplateData` function prefills a map with commonly used template data:

```go
data := newTemplateData(r, sessionManager)
err := render.Page(w, http.StatusOK, data, "your-template.tmpl")
```

Template functions are managed in the `internal/funcs` package.

## Form Validation

The application includes a comprehensive validation system with the `Validator` struct.

```go
type Validator struct {
    Errors map[string]string
}
```

`Validator` includes methods for managing errors and validation. For example:

```go
// Example ContactForm validation with Validator

type contactForm struct {
    Name    string
    Message string
    Validator
}

form := contactForm{}
form.Check(validator.NotBlank(form.Email), "Email", "Email must be a valid email address.")
form.Check(validator.NotBlank(form.Message), "Message", "Message is required.")

if form.HasErrors() { 
    // Do something with errors
}
// Do something when no errors
```

Available validators:
- `NotBlank`: Ensures string is not empty
- `MinRunes`/`MaxRunes`: Length validation
- `Between`: Range validation
- `Matches`: Regex validation
- `In`/`NotIn`: Value presence validation
- `NoDuplicates`: Uniqueness validation
- `IsEmail`: Email validation
- `IsURL`: URL validation

## Flash Messages

The application supports various flash message types. Flash messages are formatted and rendered in the `assets/templates/partials/flashMessages.tmpl` template.

```go
putFlashMessage(r, flashSuccess, "Welcome!", sessionManager)
```

Message levels:
- `flashSuccess`
- `flashError`
- `flashWarning`
- `flashInfo`

## Testing

The project includes a comprehensive testing framework with helper functions for making HTTP requests, mocking services, and asserting results.

Run tests with:

```bash
task test
```

Run tests with coverage:

```bash
task test:cover
```

## Deployment

The project includes a Dockerfile and GitHub workflow files for deployment.

## License

[MIT License](LICENSE)

## External Dependencies

- github.com/alexedwards/scs/v2
- github.com/justinas/nosurf
- github.com/wneessen/go-mail