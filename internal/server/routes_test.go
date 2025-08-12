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

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/providers"
	"github.com/gi8lino/tiledash/internal/server"
	"github.com/gi8lino/tiledash/internal/testutils"

	"github.com/stretchr/testify/assert"
)

// mockRunner implements providers.Runner.
type mockRunner struct {
	fn func(ctx context.Context) (providers.Accumulator, int, int, error)
}

func (m mockRunner) Do(ctx context.Context) (providers.Accumulator, int, int, error) {
	return m.fn(ctx)
}

func TestNewRouter(t *testing.T) {
	t.Parallel()

	// Minimal in-memory FS with required base templates + static assets.
	webFS := fstest.MapFS{
		// base templates
		"web/templates/base.gohtml":        &fstest.MapFile{Data: []byte(`{{define "base"}}<html>Tiledash>{{ template "footer" . }}</html>{{end}}`)},
		"web/templates/css/page.gohtml":    &fstest.MapFile{Data: []byte(`{{define "css_page"}}css_generic{{end}}`)},
		"web/templates/css/debug.gohtml":   &fstest.MapFile{Data: []byte(`{{define "css_debug"}}css_debug{{end}}`)},
		"web/templates/footer.gohtml":      &fstest.MapFile{Data: []byte(`{{define "footer"}}<footer>{{ .Version }}</footer>{{end}}`)},
		"web/templates/errors/page.gohtml": &fstest.MapFile{Data: []byte(`{{define "page_error"}}<!-- error -->{{end}}`)},
		"web/templates/errors/tile.gohtml": &fstest.MapFile{Data: []byte(`{{define "tile_error"}}<!-- tile error -->{{end}}`)},

		// static files
		"web/static/css/bootstrap.min.css": &fstest.MapFile{Data: []byte(`/* bootstrap */`)},
		"web/static/js/tiledash.js":        &fstest.MapFile{Data: []byte(`// js code`)},
	}

	version := "vTEST"
	debug := true
	logger := slog.New(slog.NewTextHandler(&strings.Builder{}, nil))

	// real template dir containing at least one .gohtml file used by TileHandler
	tmpDir := t.TempDir()
	testutils.MustWriteFile(t,
		filepath.Join(tmpDir, "example.gohtml"),
		// define the template with the same name the handler executes
		`{{define "example.gohtml"}}<div>{{.Title}}</div>{{end}}`,
	)

	t.Run("GET /", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{Title: "Home"}
		var runners []providers.Runner // not used by "/" handler

		router := server.NewRouter(webFS, tmpDir, cfg, logger, runners, debug, version)

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "<html>Tiledash><footer>vTEST</footer></html>", rec.Body.String())
	})

	t.Run("GET /static/css/bootstrap.min.css", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{}
		var runners []providers.Runner

		router := server.NewRouter(webFS, tmpDir, cfg, logger, runners, debug, version)

		req := httptest.NewRequest("GET", "/static/css/bootstrap.min.css", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "bootstrap")
	})

	t.Run("GET /healthz (GET)", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{}
		var runners []providers.Runner

		router := server.NewRouter(webFS, tmpDir, cfg, logger, runners, debug, version)

		req := httptest.NewRequest("GET", "/healthz", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("POST /healthz", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{}
		var runners []providers.Runner

		router := server.NewRouter(webFS, tmpDir, cfg, logger, runners, debug, version)

		req := httptest.NewRequest("POST", "/healthz", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("GET /api/v1/tile/0", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{
				{
					Template: "example.gohtml",
					Title:    "Hello",
				},
			},
		}

		// Runner returns a minimal accumulator; templates.RenderCell uses .Title from cfg.
		runners := []providers.Runner{
			mockRunner{
				fn: func(ctx context.Context) (providers.Accumulator, int, int, error) {
					return providers.Accumulator{"merged": map[string]any{"ok": true}}, 1, http.StatusOK, nil
				},
			},
		}

		router := server.NewRouter(webFS, tmpDir, cfg, logger, runners, debug, version)

		req := httptest.NewRequest("GET", "/api/v1/tile/0", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Hello")
	})

	t.Run("GET /api/v1/hash/0", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{
				{Template: "example.gohtml", Title: "Hashed"},
			},
		}
		var runners []providers.Runner

		router := server.NewRouter(webFS, tmpDir, cfg, logger, runners, debug, version)

		req := httptest.NewRequest("GET", "/api/v1/hash/0", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Regexp(t, `^[a-f0-9]+$`, rec.Body.String())
	})

	t.Run("GET /api/v1/hash/config", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{
				{Template: "example.gohtml", Title: "ConfigHash"},
			},
		}
		var runners []providers.Runner

		router := server.NewRouter(webFS, tmpDir, cfg, logger, runners, debug, version)

		req := httptest.NewRequest("GET", "/api/v1/hash/config", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Regexp(t, `^[a-f0-9]+$`, rec.Body.String())
	})
}
