package render

import (
	"bytes"
	"fmt"
	"html/template"
	"maps"
	"net/http"

	"github.com/sglmr/gowebstart/assets"
	"github.com/sglmr/gowebstart/internal/funcs"
)

// Page renders a template from the `pages` directory. It simplifies the process
// of rendering a standard page by wrapping the `PageWithHeaders` function with
// no additional headers. It takes the response writer, status code, data, and
// the name of the page template to render.
func Page(w http.ResponseWriter, status int, data any, pagePath string) error {
	return PageWithHeaders(w, status, data, nil, pagePath)
}

// PageWithHeaders renders a template from the `pages` directory with custom
// HTTP headers. It combines the base template, partials, and the specified
// page template to create a complete HTML page. This function provides more
// control over the HTTP response by allowing custom headers to be set.
func PageWithHeaders(w http.ResponseWriter, status int, data any, headers http.Header, pageName string) error {
	// Define templates to be included for this page render
	patterns := []string{"base.tmpl", "partials/*.tmpl", fmt.Sprintf("pages/%s", pageName)}

	// Render the base template with the specified patterns
	return NamedTemplateWithHeaders(w, status, data, headers, "base", patterns...)
}

// NamedTemplate renders a specific named template. It simplifies rendering by
// wrapping `NamedTemplateWithHeaders` with no additional headers. This is useful
// for rendering partial templates or templates that don't follow the standard
// page structure.
func NamedTemplate(w http.ResponseWriter, status int, data any, templateName string, patterns ...string) error {
	return NamedTemplateWithHeaders(w, status, data, nil, templateName, patterns...)
}

// NamedTemplateWithHeaders renders a specific named template with custom HTTP
// headers. It provides fine-grained control over template rendering by allowing
// you to specify which template to execute and what headers to include in the
// response. This is useful for advanced rendering scenarios.
func NamedTemplateWithHeaders(w http.ResponseWriter, status int, data any, headers http.Header, templateName string, patterns ...string) error {
	// Prepend "templates/" to all patterns to make them relative to the root
	for i := range patterns {
		patterns[i] = "templates/" + patterns[i]
	}

	// Create a new template with custom functions and parse all template files
	// from the embedded filesystem
	ts, err := template.New("").Funcs(funcs.TemplateFuncs).ParseFS(assets.EmbeddedFiles, patterns...)
	if err != nil {
		return fmt.Errorf("template.New: %w", err)
	}

	// Create a buffer to store the rendered template output
	buf := new(bytes.Buffer)

	// Execute the specified template with the provided data
	err = ts.ExecuteTemplate(buf, templateName, data)
	if err != nil {
		return fmt.Errorf("ExecuteTemplate: %w", err)
	}

	// Set any provided custom HTTP headers
	maps.Copy(w.Header(), headers)

	// Set the HTTP status code
	w.WriteHeader(status)
	buf.WriteTo(w)

	// Write the rendered template to the HTTP response
	return nil
}
