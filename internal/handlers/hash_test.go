package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gi8lino/jirapanel/internal/config"
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

		handler := HashHandler(cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Should return non-empty hex hash
		body := w.Body.String()
		assert.Regexp(t, `^[a-f0-9]+$`, body)
	})

	t.Run("returns cell hash", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Cells: []config.Cell{
				{
					Title: "Cell One",
				},
			},
		}
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/hash/0", nil)
		req.SetPathValue("id", "0")

		w := httptest.NewRecorder()

		handler := HashHandler(cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		body := w.Body.String()
		require.Equal(t, http.StatusOK, res.StatusCode)
		assert.Regexp(t, `^[a-f0-9]+$`, body)
	})

	t.Run("invalid cell id returns 400", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{}
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/hash/abc", nil)
		req.SetPathValue("id", "abc")

		w := httptest.NewRecorder()

		handler := HashHandler(cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("cell index out of bounds returns 404", func(t *testing.T) {
		t.Parallel()

		cfg := config.DashboardConfig{
			Cells: []config.Cell{},
		}
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/hash/9", nil)
		req.SetPathValue("id", "9")

		w := httptest.NewRecorder()

		handler := HashHandler(cfg, logger)
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

		handler := HashHandler(cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
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

		actual, err := hashAny(input)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("fails on non-serializable input", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int) // cannot be marshaled to JSON

		_, err := hashAny(ch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to serialize")
	})
}
