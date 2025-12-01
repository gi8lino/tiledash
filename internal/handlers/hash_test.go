package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/hash"
	"github.com/gi8lino/tiledash/internal/providers"
	"github.com/gi8lino/tiledash/internal/render"
	"github.com/gi8lino/tiledash/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns config hash", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Title: "Test Dashboard",
		}
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/hash/config", nil)
		req.SetPathValue("id", "config")

		w := httptest.NewRecorder()

		handler := HashHandler(cfg, nil, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Should return non-empty hex hash
		body := w.Body.String()
		assert.Regexp(t, `^[a-f0-9]+$`, body)
	})

	t.Run("returns tile hash", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "tile.gohtml"), []byte(`{{define "tile.gohtml"}}{{index .Data "foo"}}{{end}}`), 0o644))

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{{Title: "Cell One", Template: "tile.gohtml"}},
		}
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		funcMap := templates.TemplateFuncMap()
		cellTmpl, err := templates.ParseCellTemplates(tmpDir, funcMap)
		require.NoError(t, err)

		runners := []providers.Runner{
			mockRunner{
				fn: func(ctx context.Context) (providers.Accumulator, int, int, error) {
					return providers.Accumulator{"merged": map[string]any{"foo": "bar"}}, 1, http.StatusOK, nil
				},
			},
		}

		renderer := render.NewTileRenderer(cfg, runners, cellTmpl, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/hash/0", nil)
		req.SetPathValue("id", "0")

		w := httptest.NewRecorder()

		handler := HashHandler(cfg, renderer, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		body := strings.TrimSpace(w.Body.String())
		require.Equal(t, http.StatusOK, res.StatusCode)
		expected, herr := hash.Any("bar")
		require.NoError(t, herr)
		assert.Equal(t, expected, body)
	})

	t.Run("tile hash reflects runner data", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "tile.gohtml"), []byte(`{{define "tile.gohtml"}}{{index .Data "value"}}{{end}}`), 0o644))

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{{Title: "Cell Two", Template: "tile.gohtml"}},
		}
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		funcMap := templates.TemplateFuncMap()
		cellTmpl, err := templates.ParseCellTemplates(tmpDir, funcMap)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/hash/0", nil)
		req.SetPathValue("id", "0")

		w := httptest.NewRecorder()

		runners := []providers.Runner{
			mockRunner{
				fn: func(ctx context.Context) (providers.Accumulator, int, int, error) {
					return providers.Accumulator{"merged": map[string]any{"value": 42}}, 1, http.StatusOK, nil
				},
			},
		}

		renderer := render.NewTileRenderer(cfg, runners, cellTmpl, logger)

		handler := HashHandler(cfg, renderer, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusOK, res.StatusCode)
		expected, err := hash.Any("42")
		require.NoError(t, err)
		assert.Equal(t, expected, strings.TrimSpace(w.Body.String()))
	})

	t.Run("invalid tile id returns 400", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{}
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/hash/abc", nil)
		req.SetPathValue("id", "abc")

		w := httptest.NewRecorder()

		handler := HashHandler(cfg, nil, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("tile index out of bounds returns 404", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{Tiles: []config.Tile{}}
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		tmpDir := t.TempDir()
		funcMap := templates.TemplateFuncMap()
		cellTmpl, err := templates.ParseCellTemplates(tmpDir, funcMap)
		require.NoError(t, err)
		renderer := render.NewTileRenderer(cfg, []providers.Runner{}, cellTmpl, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/hash/9", nil)
		req.SetPathValue("id", "9")

		w := httptest.NewRecorder()

		handler := HashHandler(cfg, renderer, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("missing id returns 400", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{}
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/hash/", nil)
		// no SetPathValue

		w := httptest.NewRecorder()

		handler := HashHandler(cfg, nil, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}

type mockRunner struct {
	fn func(ctx context.Context) (providers.Accumulator, int, int, error)
}

func (m mockRunner) Do(ctx context.Context) (providers.Accumulator, int, int, error) {
	return m.fn(ctx)
}

func TestHashAny(t *testing.T) {
	t.Parallel()

	t.Run("returns correct hash for known input", func(t *testing.T) {
		t.Parallel()

		input := map[string]string{"foo": "bar"}

		// Expected hash calculated manually
		data, _ := json.Marshal(input)
		h := fnv.New64a()
		h.Write(data) // nolint:errcheck
		expected := fmt.Sprintf("%x", h.Sum64())

		actual, err := hash.Any(input)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("fails on non-serializable input", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int) // cannot be marshaled to JSON

		_, err := hash.Any(ch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to serialize")
	})
}
