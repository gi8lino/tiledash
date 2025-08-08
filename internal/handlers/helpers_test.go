package handlers

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderErrorPage(t *testing.T) {
	t.Parallel()

	t.Run("renders HTML error page", func(t *testing.T) {
		t.Parallel()

		// Define a base template with an "error" template block
		tmpl := template.Must(template.New("base").Parse(`
			{{define "cell_error"}}<html><head><title>{{.Title}}</title></head><body>
			<h1>{{.Message}}</h1><pre>{{.Error}}</pre></body></html>{{end}}
		`))

		rr := httptest.NewRecorder()

		renderErrorPage(rr, http.StatusInternalServerError, tmpl, "Internal Error", "Something went wrong", errors.New("example failure"))

		resp := rr.Result()
		defer resp.Body.Close() // nolint:errcheck

		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		}

		body := rr.Body.String()

		expected := `<html><head><title>Internal Error</title></head><body>
			<h1>Something went wrong</h1><pre>example failure</pre></body></html>`
		require.Equal(t, expected, body)
	})

	t.Run("template execution fails gracefully", func(t *testing.T) {
		t.Parallel()

		// Intentionally omit the "error" block to trigger ExecuteTemplate error
		tmpl := template.Must(template.New("base").Parse(`<html><body>{{.Unused}}</body></html>`))

		rr := httptest.NewRecorder()

		renderErrorPage(rr, http.StatusNotFound, tmpl, "Not Found", "Missing template", errors.New("missing"))

		resp := rr.Result()
		defer resp.Body.Close() // nolint:errcheck

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, "<div class=\"alert alert-danger\">Failed to render error page</div>", rr.Body.String())
	})
}
