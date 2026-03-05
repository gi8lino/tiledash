package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRoutePrefix(t *testing.T) {
	t.Parallel()

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "", NormalizeRoutePrefix(""))
		require.Equal(t, "", NormalizeRoutePrefix("   "))
		require.Equal(t, "", NormalizeRoutePrefix("/"))
	})

	t.Run("simple paths", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "/tiledash", NormalizeRoutePrefix("tiledash"))
		require.Equal(t, "/tiledash", NormalizeRoutePrefix("/tiledash"))
		require.Equal(t, "/tiledash", NormalizeRoutePrefix("/tiledash/"))
		require.Equal(t, "/tiledash", NormalizeRoutePrefix("   /tiledash/   "))
	})

	t.Run("multiple trailing slashes(", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "/api", NormalizeRoutePrefix("/api///"))
	})

	t.Run("full URL with path", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "/tiledash", NormalizeRoutePrefix("https://example.com/tiledash"))
		require.Equal(t, "/tiledash", NormalizeRoutePrefix("https://example.com/tiledash/"))
	})

	t.Run("full URL with no path", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "", NormalizeRoutePrefix("https://example.com"))
		require.Equal(t, "", NormalizeRoutePrefix("https://example.com/"))
	})

	t.Run("malformed URL treated as path", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "/://bad-url", NormalizeRoutePrefix("://bad-url"))
	})

	t.Run("root like input", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "", NormalizeRoutePrefix("///"))
	})
}

func TestMountUnderPrefix(t *testing.T) {
	t.Parallel()

	inner := http.NewServeMux()
	inner.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "root")
	})
	inner.HandleFunc("GET /foo", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "foo")
	})
	inner.HandleFunc("GET /api/v1/ok", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})

	t.Run("prefix '/' behaves like root", func(t *testing.T) {
		t.Parallel()

		h := mountUnderPrefix(inner, "/")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		h.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "foo", rec.Body.String()) // <- correct
	})

	t.Run("prefix without leading slash is normalized", func(t *testing.T) {
		h := mountUnderPrefix(inner, "tiledash")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tiledash/foo", nil)
		h.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "foo", rec.Body.String())
	})

	t.Run("prefix with trailing slash is normalized", func(t *testing.T) {
		h := mountUnderPrefix(inner, "/tiledash/")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tiledash/foo", nil)
		h.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "foo", rec.Body.String())
	})

	t.Run("empty prefix returns original handler (serves at root)", func(t *testing.T) {
		t.Parallel()

		h := mountUnderPrefix(inner, "")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "root", rec.Body.String())

		rec2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/tiledash/foo", nil)
		h.ServeHTTP(rec2, req2)
		require.Equal(t, http.StatusOK, rec2.Code)
		require.Equal(t, "root", rec2.Body.String())
	})

	t.Run("bare prefix GET redirects with 308", func(t *testing.T) {
		t.Parallel()

		h := mountUnderPrefix(inner, "/tiledash")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tiledash", nil)
		h.ServeHTTP(rec, req)

		require.Equal(t, http.StatusPermanentRedirect, rec.Code) // 308
		assert.Equal(t, "/tiledash/", rec.Header().Get("Location"))
	})

	t.Run("bare prefix HEAD redirects with 308", func(t *testing.T) {
		t.Parallel()

		h := mountUnderPrefix(inner, "/tiledash")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodHead, "/tiledash", nil)
		h.ServeHTTP(rec, req)

		require.Equal(t, http.StatusPermanentRedirect, rec.Code) // 308
		assert.Equal(t, "/tiledash/", rec.Header().Get("Location"))
	})

	t.Run("POST to bare prefix redirects with 307 (method preserved)", func(t *testing.T) {
		t.Parallel()

		h := mountUnderPrefix(inner, "/tiledash")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/tiledash", nil)
		h.ServeHTTP(rec, req)

		require.Equal(t, http.StatusTemporaryRedirect, rec.Code) // 307
		assert.Equal(t, "/tiledash/", rec.Header().Get("Location"))
	})

	t.Run("prefixed paths are stripped and routed to inner handler", func(t *testing.T) {
		t.Parallel()

		h := mountUnderPrefix(inner, "/tiledash")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tiledash/foo", nil)
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "foo", rec.Body.String())

		rec2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/tiledash/api/v1/ok", nil)
		h.ServeHTTP(rec2, req2)
		require.Equal(t, http.StatusOK, rec2.Code)
		require.Equal(t, "ok", rec2.Body.String())
	})

	t.Run("prefix with trailing slash serves inner root", func(t *testing.T) {
		t.Parallel()

		h := mountUnderPrefix(inner, "/tiledash")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tiledash/", nil)
		h.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "root", rec.Body.String())
	})

	t.Run("non-prefixed paths 404 when mounted under a prefix", func(t *testing.T) {
		t.Parallel()

		h := mountUnderPrefix(inner, "/tiledash")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
