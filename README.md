# Go Web Starter

A lightweight, feature-rich Go web application template with built-in security features, session management, database integration, email functionality, authentication, and more.

This project serves as a robust starting point for modern web applications in Go. It comes pre-configured with a SQLite database, database migrations, and type-safe queries via SQLC.

This project assumes you will be running it behind a reverse proxy service that handles HTTPS and certificates for you.

## Features

- **HTTP Server**: A robust HTTP server with graceful shutdown.
- **Routing**: Fast and flexible routing with [chi](https://github.com/go-chi/chi).
- **Database Integration**: Comes with a SQLite database, [golang-migrate](https://github.com/golang-migrate/migrate) for migrations, and [SQLC](https://sqlc.dev/) for type-safe SQL queries.
- **Authentication System**: Login/Logout functionality with session management.
- **Middleware Stack**:
  - Panic recovery
  - Secure headers
  - Request logging
  - CSRF protection
  - Session management
- **Email Support**: Send emails with configurable SMTP or a logger backend for development.
- **Form Validation**: Comprehensive validation helpers.
- **Flash Messages**: Session-based notification system.
- **Templating**: HTML template rendering with data context.
- **TailwindCSS**: Style HTML pages with TailwindCSS.
- **Static File Serving**: Embedded static file handling.
- **Development Mode**: Enhanced debugging with stack traces and additional logging.
- **Live Reload**: Live reload with [air](https://github.com/air-verse/air).
- **Task Runner**: Simple and efficient task management with [Task](https://taskfile.dev/).

## Getting Started

### Prerequisites

- Go 1.24 or higher
- [Task](https://taskfile.dev/) for project management commands
- [Air](https://github.com/air-verse/air) for live reload
- [Node.js](https://nodejs.org/en) and npm for Tailwind CSS

### Installation

1.  Clone the repository:

    ```bash
    git clone https://github.com/sglmr/go-web-starter.git
    cd go-web-starter
    ```

2.  Install Go and JavaScript dependencies:

    ```bash
    go mod tidy
    npm install
    ```

3.  Create the database and run migrations:

    ```bash
    task migrate:up
    ```

4.  Run the application to make sure everything is working:

    ```bash
    task run:live
    ```

5.  Replace `"github.com/sglmr/gowebstart"` with your new project name across the project.

### Running the Server

The project uses `task` to simplify common commands.

| Command | Description |
| :--- | :--- |
| `task run:live` | Run the app with live reload (rebuilds CSS automatically). |
| `task run` | Run the app without live reload. |
| `task build` | Build the application binary. |
| `task test` | Run all tests. |
| `task test:cover` | Run tests and view coverage. |
| `task tailwind:watch` | Watch for CSS changes and rebuild `main.css`. |

The server will start on `127.0.0.1:8000` by default.

### Command-Line Options

| Flag | Environment Variable | Description | Default |
| :--- | :--- | :--- | :--- |
| `-host` | | Server host | `127.0.0.1` |
| `-port` | `PORT` | Server port | `8000` |
| `-dev` | | Development mode | `false` |
| `-db-path` | `DB_PATH` | Path to SQLite database | `tmp/db.sqlite` |
| `-send-email` | | Send live emails | `false` |
| `-smtp-host` | `SMTP_HOST` | SMTP server host | |
| `-smtp-port` | `SMTP_PORT` | SMTP server port | |
| `-smtp-username` | `SMTP_USERNAME` | SMTP username | |
| `-smtp-password` | `SMTP_PASSWORD` | SMTP password | |
| `-smtp-from` | `SMTP_FROM` | Email sender | |

Example with custom options:

```bash
./web -port=3000 -dev -db-path=./my-app.db
```

## Database

The project is configured to use a SQLite database.

### Migrations

Database schema migrations are managed with `golang-migrate`.

-   **Create a new migration:**

    ```bash
    task migrate:new create_products_table
    ```

    This will create `up` and `down` migration files in `internal/data/migrations`.

-   **Run migrations:**

    ```bash
    task migrate:up
    ```

### SQLC

[SQLC](https://sqlc.dev/) generates type-safe Go code from your SQL queries. The configuration is in `sqlc.yaml`.

-   Write your SQL queries in `internal/data/queries/`.
-   Generate Go code with:

    ```bash
    task sqlc:generate
    ```

    This will generate/update files in `internal/data/` based on your queries.

## Authentication

The template includes a login/logout system for users.

-   **Login page**: `http://localhost:8000/login`
-   **Logout page**: `http://localhost:8000/logout`

Protected routes can be set up using the `requireLogin` middleware.

### Creating Password Hashes

You can use the included `hash` tool to generate secure password hashes:

```sh
go run ./cmd/hash
```

## Architecture

-   `assets/`: Embedded project files (templates, static assets, etc.).
-   `cmd/`: Application entry points.
    -   `web/`: Main web server application.
    -   `hash/`: CLI tool for hashing passwords.
-   `internal/`: Internal packages for the application's business logic.
    -   `data/`: Database logic, including models, migrations, and SQLC-generated code.
    -   `email/`: SMTP email functionality.
    -   `render/`: Template rendering helpers.
    -   `validator/`: Form validation.
    -   `web/`: Web-related functionality, including routing, middleware, and handlers.
-   `.air.toml`: Live reload configuration.
-   `Taskfile.yml`: Project tasks.
-   `sqlc.yaml`: SQLC configuration.

## Deployment

The project includes a `Dockerfile` for building a minimal container image and GitHub Actions workflows in `.github/workflows` for CI/DE.

## External Dependencies

### Go
- github.com/alexedwards/scs/v2
- github.com/go-chi/chi
- github.com/golang-migrate/migrate/v4
- github.com/justinas/nosurf
- github.com/wneessen/go-mail
- github.com/sqlc-dev/sqlc

### npm
- tailwindcss
- @tailwindcss/forms
- @tailwindcss/typography

## License

[MIT License](LICENSE)
