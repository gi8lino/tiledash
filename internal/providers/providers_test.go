package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRegistry(t *testing.T) {
	// NOTE: don't call t.Parallel() here since we’ll spin up servers in subtests.

	t.Run("BuildRegistry success and BuildRunners happy path", func(t *testing.T) {
		// Each subtest gets its own server and cleanup.
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		t.Cleanup(ts.Close)

		provs := map[string]config.Provider{
			"Jira-V2": {BaseURL: ts.URL}, // mixed case name
		}

		reg, err := BuildRegistry(provs)
		require.NoError(t, err)
		require.Len(t, reg, 1)

		_, ok := reg["jira-v2"]
		assert.True(t, ok, "provider key should be lower-cased")

		tiles := []config.Tile{
			{
				Title: "T",
				Request: config.Request{
					Provider: "jira-v2",
					Method:   http.MethodGet,
					Path:     "/",
				},
			},
		}

		runners, err := BuildRunners(reg, tiles)
		require.NoError(t, err)
		require.Len(t, runners, 1)

		acc, pages, status, err := runners[0].Do(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, pages)
		assert.Equal(t, http.StatusOK, status)
		require.NotNil(t, acc)
	})

	t.Run("BuildRegistry error on invalid baseURL", func(t *testing.T) {
		_, err := BuildRegistry(map[string]config.Provider{
			"bad": {BaseURL: "://bad"},
		})
		require.Error(t, err)
	})
}

func TestCompile(t *testing.T) {
	// Independent subtests; each can run in parallel safely.

	t.Run("compiles valid provider (trims and lowercases name)", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		t.Cleanup(ts.Close)

		provs := map[string]config.Provider{
			"Jira-V2": {BaseURL: ts.URL},
		}
		reg, err := BuildRegistry(provs)
		require.NoError(t, err)

		r, err := reg.compile(config.Request{Provider: "  JIRA-v2  "})
		require.NoError(t, err)
		require.NotNil(t, r)

		hr, ok := r.(*HTTPRunner)
		require.True(t, ok, "runner should be *HTTPRunner")
		require.NotNil(t, hr.prov)
		assert.Equal(t, "jira-v2", hr.prov.Name)
	})

	t.Run("unknown provider", func(t *testing.T) {
		t.Parallel()

		reg := Registry{} // empty
		_, err := reg.compile(config.Request{Provider: "missing"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown provider "missing"`)
	})

	t.Run("empty provider name", func(t *testing.T) {
		t.Parallel()

		reg := Registry{} // empty
		_, err := reg.compile(config.Request{Provider: ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown provider ""`)
	})
}

func TestBuildRunners_ErrorWrapsTileInfo(t *testing.T) {
	t.Parallel()

	reg := Registry{} // no providers registered

	tiles := []config.Tile{
		{Title: "First", Request: config.Request{Provider: "missing"}},
	}

	_, err := BuildRunners(reg, tiles)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `tile 0 (First): unknown provider "missing"`)
}
