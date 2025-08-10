package handlers

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/providers"
	"github.com/gi8lino/tiledash/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	acc    providers.Accumulator
	pages  int
	status int
	err    error
}

func (f fakeRunner) Do(ctx context.Context) (providers.Accumulator, int, int, error) {
	return f.acc, f.pages, f.status, f.err
}

func TestTileHandler(t *testing.T) {
	t.Parallel()

	t.Run("renders tile successfully", func(t *testing.T) {
		t.Parallel()

		// error template used by renderCellError
		webFS := fstest.MapFS{
			"web/templates/errors/tile.gohtml": &fstest.MapFile{
				Data: []byte(`{{define "tile_error"}}ERROR: {{.Message}}{{end}}`),
			},
		}

		// a real tile template the handler will load from disk
		tmpDir := t.TempDir()
		testutils.MustWriteFile(t,
			filepath.Join(tmpDir, "tile.gohtml"),
			`{{define "tile.gohtml"}}<div>Cell: {{.Title}} / {{ index .Data "key" }}</div>{{end}}`,
		)

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{
				{
					Title:    "Test Cell",
					Template: "tile.gohtml",
				},
			},
			RefreshInterval: 5 * time.Second,
		}

		// runner returns a single-page JSON object; normalizeData exposes it under .Data
		runners := []providers.Runner{
			fakeRunner{
				acc:   providers.Accumulator(map[string]any{"key": "value"}),
				pages: 1, status: http.StatusOK,
			},
		}

		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tile/0", nil)
		req.SetPathValue("id", "0")

		w := httptest.NewRecorder()

		h := TileHandler(webFS, tmpDir, "vX", cfg, runners, logger)
		h.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(body), "Cell: Test Cell / value")
	})

	t.Run("renders error on upstream failure", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/errors/tile.gohtml": &fstest.MapFile{
				Data: []byte(`{{define "tile_error"}}<div class="error">Error: {{.Message}}</div>{{end}}`),
			},
		}

		tmpDir := t.TempDir()
		testutils.MustWriteFile(t, filepath.Join(tmpDir, "dummy.gohtml"),
			`{{define "dummy.gohtml"}}noop{{end}}`,
		)

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{
				{Title: "Broken", Template: "dummy.gohtml"},
			},
		}

		runners := []providers.Runner{
			fakeRunner{
				err:    assert.AnError,
				status: http.StatusInternalServerError,
			},
		}

		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tile/0", nil)
		req.SetPathValue("id", "0")

		w := httptest.NewRecorder()

		h := TileHandler(webFS, tmpDir, "dev", cfg, runners, logger)
		h.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		require.Equal(t, http.StatusBadGateway, res.StatusCode)

		// The handler sets Message="request failed"
		assert.Equal(t, `<div class="error">Error: request failed</div>`, strings.TrimSpace(string(body)))
	})

	t.Run("invalid id returns error", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/errors/tile.gohtml": &fstest.MapFile{
				Data: []byte(`{{define "tile_error"}}Message: {{.Message}}{{end}}`),
			},
		}

		cfg := config.DashboardConfig{}
		var runners []providers.Runner // empty -> any index is invalid

		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/layout/foo", nil)
		req.SetPathValue("id", "foo")

		w := httptest.NewRecorder()

		h := TileHandler(webFS, ".", "x", cfg, runners, logger)
		h.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck
		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "Message: Invalid tile id", string(body))
	})

	t.Run("missing id returns error", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/errors/tile.gohtml": &fstest.MapFile{
				Data: []byte(`{{define "tile_error"}}ERROR: {{.Message}}{{end}}`),
			},
		}
		cfg := config.DashboardConfig{}
		var runners []providers.Runner

		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/tile/", nil)
		// no SetPathValue => missing id

		w := httptest.NewRecorder()

		h := TileHandler(webFS, ".", "x", cfg, runners, logger)
		h.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}
