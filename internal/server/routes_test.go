package server_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/server"

	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	// Minimal in-memory file system
	webFS := fstest.MapFS{
		// templates
		"web/templates/base.gohtml":   &fstest.MapFile{Data: []byte(`{{define "base"}}<html>Jira Panel>{{ template "footer" . }}</html>{{end}}`)},
		"web/templates/footer.gohtml": &fstest.MapFile{Data: []byte(`{{define "footer"}}<footer>{{ .Version }}</footer>{{end}}`)},
		"web/templates/error.gohtml":  &fstest.MapFile{Data: []byte(`{{define "error"}}<!-- error -->{{end}}`)},

		// static files
		"web/static/css/bootstrap.min.css": &fstest.MapFile{Data: []byte(`/* bootstrap */`)},
		"web/static/js/jirapanel.js":       &fstest.MapFile{Data: []byte(`// js code`)},
	}

	// Dummy dependencies
	templateDir := "/web/templates"
	version := "vTEST"
	debug := true
	logger := slog.New(slog.NewTextHandler(&strings.Builder{}, nil))

	// Minimal valid config
	cfg := config.DashboardConfig{}

	// Dummy Jira client
	client := &jira.Client{}

	// Build the router
	router := server.NewRouter(webFS, templateDir, client, cfg, logger, debug, version)

	t.Run("GET /", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "<html>Jira Panel><footer>vTEST</footer></html>", rec.Body.String())
	})

	t.Run("GET /static/css/bootstrap.min.css", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/css/bootstrap.min.css", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "bootstrap")
	})

	t.Run("GET /healthz", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("POST /healthz", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/healthz", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})
}
