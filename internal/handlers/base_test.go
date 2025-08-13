package handlers

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboard(t *testing.T) {
	t.Parallel()

	t.Run("renders dashboard successfully", func(t *testing.T) {
		t.Parallel()

		// Base template set used by BaseHandler
		webFS := fstest.MapFS{
			"web/templates/base.gohtml":        &fstest.MapFile{Data: []byte(`{{define "base"}}{{.Title}} v{{.Version}}{{end}}`)},
			"web/templates/css/page.gohtml":    &fstest.MapFile{Data: []byte(`{{define "css_page"}}css_generic{{end}}`)},
			"web/templates/css/debug.gohtml":   &fstest.MapFile{Data: []byte(`{{define "css_debug"}}css_debug{{end}}`)},
			"web/templates/footer.gohtml":      &fstest.MapFile{Data: []byte(`{{define "footer"}}footer{{end}}`)},
			"web/templates/errors/page.gohtml": &fstest.MapFile{Data: []byte(`{{define "page_error"}}Error: {{.Message}}{{end}}`)},
		}

		// Create a tile template directory to pass to BaseHandler (not directly used by BaseHandler,
		// but we keep it to mirror your runtime layout).
		tmpDir := t.TempDir()
		testutils.MustWriteFile(t,
			filepath.Join(tmpDir, "templates", "tile_example.gohtml"),
			`{{define "tile_example.html"}}<div>{{.Title}}</div>{{end}}`,
		)

		cfg := config.DashboardConfig{
			Title: "My Dashboard",
			Grid:  &config.GridConfig{Columns: 3, Rows: 2},
			Tiles: []config.Tile{
				{
					Title:    "Example Section",
					Template: "tile_example.html",
					// Position values are not used by BaseHandler rendering itself
					Position: config.Position{Row: 1, Col: 1},
				},
			},
			RefreshInterval: 30 * time.Second,
		}

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler := BaseHandler(webFS, filepath.Join(tmpDir, "templates"), "", "1.0.0", cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck
		body := w.Body.String()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, body, "My Dashboard v1.0.0")
	})

	t.Run("renders page_error when base template execution fails", func(t *testing.T) {
		t.Parallel()

		// Intentionally reference a missing template inside base.gohtml to force ExecuteTemplate error.
		webFS := fstest.MapFS{
			"web/templates/base.gohtml":        &fstest.MapFile{Data: []byte(`{{define "base"}}{{template "missing_subtemplate"}}{{end}}`)},
			"web/templates/css/page.gohtml":    &fstest.MapFile{Data: []byte(`{{define "css_page"}}css_generic{{end}}`)},
			"web/templates/css/debug.gohtml":   &fstest.MapFile{Data: []byte(`{{define "css_debug"}}css_debug{{end}}`)},
			"web/templates/footer.gohtml":      &fstest.MapFile{Data: []byte(`{{define "footer"}}footer{{end}}`)},
			"web/templates/errors/page.gohtml": &fstest.MapFile{Data: []byte(`{{define "page_error"}}Error: {{.Message}}{{end}}`)},
		}

		tmpDir := t.TempDir()
		testutils.MustWriteFile(t,
			filepath.Join(tmpDir, "templates", "dummy.gohtml"),
			`{{define "dummy"}}noop{{end}}`,
		)

		cfg := config.DashboardConfig{
			Title:           "Broken Dashboard",
			RefreshInterval: 10 * time.Second,
			Tiles: []config.Tile{
				{Title: "Failing Section", Template: "dummy"},
			},
		}

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler := BaseHandler(webFS, filepath.Join(tmpDir, "templates"), "", "dev", cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		// BaseHandler uses renderErrorPage with StatusInternalServerError on template failure.
		require.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Equal(t, "Error: Failed to render dashboard tiles.", string(bytes.TrimSpace(body)))
	})
}

func TestComputeCellHashes(t *testing.T) {
	t.Parallel()

	t.Run("computes hashes for all tiles", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{
				{
					Title:    "First",
					Template: "simple.gohtml",
					Position: config.Position{Row: 1, Col: 1},
				},
				{
					Title:    "Second",
					Template: "detailed.gohtml",
					Position: config.Position{Row: 2, Col: 1},
				},
			},
		}

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		computeCellHashes(&cfg, logger)

		require.NotEmpty(t, cfg.Tiles[0].Hash)
		require.NotEmpty(t, cfg.Tiles[1].Hash)
		assert.NotEqual(t, cfg.Tiles[0].Hash, cfg.Tiles[1].Hash)
	})
}
