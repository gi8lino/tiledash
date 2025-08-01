package handlers

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderErrorPage(t *testing.T) {
	t.Parallel()

	t.Run("renders HTML error page", func(t *testing.T) {
		t.Parallel()

		// Define a base template with an "error" template block
		tmpl := template.Must(template.New("base").Parse(`
			{{define "error"}}<html><head><title>{{.Title}}</title></head><body>
			<h1>{{.Message}}</h1><pre>{{.Error}}</pre></body></html>{{end}}
		`))

		rr := httptest.NewRecorder()

		renderErrorPage(rr, http.StatusInternalServerError, tmpl, "Internal Error", "Something went wrong", errors.New("example failure"))

		resp := rr.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		}

		body := rr.Body.String()

		if !strings.Contains(body, "<title>Internal Error</title>") {
			t.Errorf("missing title in output: %q", body)
		}
		if !strings.Contains(body, "<h1>Something went wrong</h1>") {
			t.Errorf("missing message in output: %q", body)
		}
		if !strings.Contains(body, "<pre>example failure</pre>") {
			t.Errorf("missing error detail in output: %q", body)
		}
	})

	t.Run("template execution fails gracefully", func(t *testing.T) {
		t.Parallel()

		// Intentionally omit the "error" block to trigger ExecuteTemplate error
		tmpl := template.Must(template.New("base").Parse(`<html><body>{{.Unused}}</body></html>`))

		rr := httptest.NewRecorder()

		renderErrorPage(rr, http.StatusNotFound, tmpl, "Not Found", "Missing template", errors.New("missing"))

		resp := rr.Result()
		defer resp.Body.Close() // nolint:errcheck

		// Status code should still be correct
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
		}

		// Since ExecuteTemplate fails, the body will likely be empty
		if rr.Body.Len() == 0 {
			// Acceptable fallback behavior
			return
		}
	})
}
