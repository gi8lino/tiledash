package server_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/server"
	"github.com/gi8lino/jirapanel/internal/testutils"

	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	// Minimal in-memory file system
	webFS := fstest.MapFS{
		// templates
		"web/templates/base.gohtml":        &fstest.MapFile{Data: []byte(`{{define "base"}}<html>Jira Panel>{{ template "footer" . }}</html>{{end}}`)},
		"web/templates/css_generic.gohtml": &fstest.MapFile{Data: []byte(`{{define "css_generic"}}css_generic{{end}}`)},
		"web/templates/css_debug.gohtml":   &fstest.MapFile{Data: []byte(`{{define "css_debug"}}css_debug{{end}}`)},
		"web/templates/footer.gohtml":      &fstest.MapFile{Data: []byte(`{{define "footer"}}<footer>{{ .Version }}</footer>{{end}}`)},
		"web/templates/error.gohtml":       &fstest.MapFile{Data: []byte(`{{define "error"}}<!-- error -->{{end}}`)},
		"web/templates/cell_error.gohtml":  &fstest.MapFile{Data: []byte(`{{define "cell_error"}}<!-- cell error -->{{end}}`)},

		// static files
		"web/static/css/bootstrap.min.css": &fstest.MapFile{Data: []byte(`/* bootstrap */`)},
		"web/static/js/jirapanel.js":       &fstest.MapFile{Data: []byte(`// js code`)},
	}

	// Dummy dependencies
	version := "vTEST"
	debug := true
	logger := slog.New(slog.NewTextHandler(&strings.Builder{}, nil))

	// Dummy Jira client
	client := &jira.Client{}

	// Create real cell templates dir with at least one .gohtml file
	tmpDir := t.TempDir()
	testutils.MustWriteFile(t,
		filepath.Join(tmpDir, "example.gohtml"),
		`<div>{{.Title}}</div>`,
	)

	t.Run("GET /", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		// Minimal valid config
		cfg := config.DashboardConfig{}

		// Build the router
		router := server.NewRouter(webFS, tmpDir, client, cfg, logger, debug, version)

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "<html>Jira Panel><footer>vTEST</footer></html>", rec.Body.String())
	})

	t.Run("GET /static/css/bootstrap.min.css", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/css/bootstrap.min.css", nil)
		rec := httptest.NewRecorder()

		// Minimal valid config
		cfg := config.DashboardConfig{}

		// Build the router
		router := server.NewRouter(webFS, tmpDir, client, cfg, logger, debug, version)

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "bootstrap")
	})

	t.Run("GET /healthz", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		rec := httptest.NewRecorder()

		// Minimal valid config
		cfg := config.DashboardConfig{}

		// Build the router
		router := server.NewRouter(webFS, tmpDir, client, cfg, logger, debug, version)

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("POST /healthz", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/healthz", nil)
		rec := httptest.NewRecorder()

		// Minimal valid config
		cfg := config.DashboardConfig{}

		// Build the router
		router := server.NewRouter(webFS, tmpDir, client, cfg, logger, debug, version)

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("GET /cell/0", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Cells: []config.Cell{
				{
					Template: "example.gohtml",
					Title:    "Hello",
				},
			},
		}

		mockClient := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{"issues":[{"fields":{"summary":"Mock issue"}}]}`), 200, nil
			},
		}

		router := server.NewRouter(webFS, tmpDir, mockClient, cfg, logger, debug, version)

		req := httptest.NewRequest("GET", "/api/v1/cell/0", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Hello")
	})

	t.Run("GET /hash/0", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Cells: []config.Cell{
				{
					Template: "example", // will transormed to "example.html"
					Title:    "Hashed",
				},
			},
		}
		router := server.NewRouter(webFS, tmpDir, client, cfg, logger, debug, version)

		req := httptest.NewRequest("GET", "/api/v1/hash/0", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Regexp(t, `^[a-f0-9]+$`, rec.Body.String()) // hex hash
	})

	t.Run("GET /hash/config", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Cells: []config.Cell{
				{
					Template: "example.html",
					Title:    "ConfigHash",
				},
			},
		}
		router := server.NewRouter(webFS, tmpDir, client, cfg, logger, debug, version)

		req := httptest.NewRequest("GET", "/api/v1/hash/config", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Regexp(t, `^[a-f0-9]+$`, rec.Body.String()) // hex hash
	})
}
