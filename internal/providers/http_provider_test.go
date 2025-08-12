package providers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPProvider(t *testing.T) {
	t.Parallel()

	t.Run("missing baseURL", func(t *testing.T) {
		t.Parallel()
		_, err := NewHTTPProvider("p1", config.Provider{BaseURL: ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `provider "p1": missing baseURL`)
	})

	t.Run("invalid baseURL", func(t *testing.T) {
		t.Parallel()
		_, err := NewHTTPProvider("p1", config.Provider{BaseURL: ":// bad"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `provider "p1": invalid baseURL`)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		p, err := NewHTTPProvider("p1", config.Provider{BaseURL: "http://example.com"})
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, "p1", p.Name)
		assert.Equal(t, "http://example.com", p.Base.String())
		require.NotNil(t, p.Client)
		require.NotNil(t, p.Cache)
	})
}

func TestRunner_NonPaginated_JSONBody_AndCache(t *testing.T) {
	t.Parallel()

	t.Run("Non paginated JSON body and cache", func(t *testing.T) {
		t.Parallel()

		var hits int
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			b, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":  true,
				"ct":  r.Header.Get("Content-Type"),
				"bdy": string(b),
			})
		}))
		t.Cleanup(ts.Close)

		p, err := NewHTTPProvider("p", config.Provider{BaseURL: ts.URL})
		require.NoError(t, err)

		r := p.NewRunner(config.Request{
			Provider: "p",
			Method:   http.MethodPost,
			Path:     "/anything",
			TTL:      500 * time.Millisecond,
			BodyJSON: map[string]any{"a": 1},
			Headers:  map[string]string{"X-Req": "1"}, // no Content-Type on purpose
		})

		// 1st call -> upstream hit
		acc, pages, status, err := r.Do(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 1, pages)
		assert.Equal(t, http.StatusOK, status)

		// read from first page (not merged)
		var page0 map[string]any
		switch ps := acc["pages"].(type) {
		case []map[string]any:
			require.NotEmpty(t, ps)
			page0 = ps[0]
		case []any:
			require.NotEmpty(t, ps)
			page0, _ = ps[0].(map[string]any)
		default:
			t.Fatalf("unexpected pages type: %T", ps)
		}
		require.NotNil(t, page0)
		assert.Equal(t, true, page0["ok"])
		assert.Equal(t, "application/json", page0["ct"])
		assert.Contains(t, page0["bdy"].(string), `"a":1`)

		// 2nd call -> cache hit, upstream still 1
		_, _, _, err = r.Do(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 1, hits, "expected cache hit on second call")
	})
}

func TestRunner_NonPaginated_RawBody_AndHeaderOverride(t *testing.T) {
	t.Parallel()

	t.Run("raw body respected", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ct":  r.Header.Get("Content-Type"),
				"bdy": string(b),
				"m":   r.Method,
			})
		}))
		t.Cleanup(ts.Close)

		p, err := NewHTTPProvider("p", config.Provider{BaseURL: ts.URL})
		require.NoError(t, err)

		r := p.NewRunner(config.Request{
			Provider: "p",
			Method:   http.MethodPut,
			Path:     "/raw",
			TTL:      0,
			Body:     "hello",
			Headers:  map[string]string{"Content-Type": "text/plain"},
		})
		acc, _, _, err := r.Do(t.Context())
		require.NoError(t, err)

		page0 := firstPage(t, acc)
		assert.Equal(t, "text/plain", page0["ct"])
		assert.Equal(t, "hello", page0["bdy"])
		assert.Equal(t, "PUT", page0["m"])
	})

	t.Run("BodyJSON but Content-Type preset -> do not override", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ct":  r.Header.Get("Content-Type"),
				"bdy": string(b),
			})
		}))
		t.Cleanup(ts.Close)

		p, err := NewHTTPProvider("p", config.Provider{BaseURL: ts.URL})
		require.NoError(t, err)

		r := p.NewRunner(config.Request{
			Provider: "p",
			Method:   http.MethodPost,
			Path:     "/json",
			TTL:      0,
			BodyJSON: map[string]any{"x": 1},
			Headers:  map[string]string{"Content-Type": "application/foo"},
		})
		acc, _, _, err := r.Do(t.Context())
		require.NoError(t, err)

		page0 := firstPage(t, acc)
		assert.Equal(t, "application/foo", page0["ct"])
		assert.Contains(t, page0["bdy"].(string), `"x":1`)
	})
}

// small helper to read first page robustly
func firstPage(t *testing.T, acc Accumulator) map[string]any {
	t.Helper()
	switch ps := acc["pages"].(type) {
	case []map[string]any:
		require.NotEmpty(t, ps)
		return ps[0]
	case []any:
		require.NotEmpty(t, ps)
		m, _ := ps[0].(map[string]any)
		require.NotNil(t, m)
		return m
	default:
		t.Fatalf("unexpected pages type: %T", ps)
		return nil
	}
}

func TestRunner_Paginated_QueryParams(t *testing.T) {
	t.Parallel()

	// Simulate a dataset of 5 issues; pagination via query s,l
	data := []int{1, 2, 3, 4, 5}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		s, _ := url.QueryUnescape(q.Get("s"))
		l, _ := url.QueryUnescape(q.Get("l"))
		// parse ints, default 0/2 for robustness in this test
		start, limit := 0, 2
		if s != "" {
			start = testutils.AtoiSafe(s)
		}
		if l != "" {
			limit = testutils.AtoiSafe(l)
		}
		end := start + limit
		if end > len(data) {
			end = len(data)
		}
		issues := make([]map[string]any, 0, end-start)
		for _, v := range data[start:end] {
			issues = append(issues, map[string]any{"id": v})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"start":  start,
			"limit":  limit,
			"total":  len(data),
			"issues": issues,
		})
	}))
	defer ts.Close()

	p, _ := NewHTTPProvider("p", config.Provider{BaseURL: ts.URL})

	r := p.NewRunner(config.Request{
		Provider: "p",
		Method:   http.MethodGet,
		Path:     "/items",
		Paginate: true,
		Page: config.PageParams{
			Location:   "query",
			StartField: "start",
			LimitField: "limit",
			TotalField: "total",
			ReqStart:   "s",
			ReqLimit:   "l",
		},
		// default query e.g. filter=abc not needed here
	})

	acc, pages, status, err := r.Do(t.Context())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 3, pages, "5 items with limit=2 -> 3 pages")

	merged := acc["merged"].(map[string]any)
	issAny := merged["issues"]
	switch v := issAny.(type) {
	case []map[string]any:
		require.Len(t, v, 5)
	case []any:
		require.Len(t, v, 5)
	default:
		t.Fatalf("unexpected issues type: %T", v)
	}
}

func TestRunner_Paginated_Body(t *testing.T) {
	t.Parallel()

	// echo pagination from JSON body fields "startAt","maxResults"
	data := []int{10, 11, 12}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = r.Body.Close()

		start := testutils.AtoiAny(body["startAt"])
		limit := testutils.AtoiAny(body["maxResults"])
		if limit <= 0 {
			limit = 1
		}
		end := start + limit
		end = min(end, len(data))
		items := make([]map[string]any, 0, end-start)
		for _, v := range data[start:end] {
			items = append(items, map[string]any{"id": v})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"startAt":    start,
			"maxResults": limit,
			"total":      len(data),
			"items":      items,
		})
	}))
	defer ts.Close()

	p, _ := NewHTTPProvider("p", config.Provider{BaseURL: ts.URL})

	r := p.NewRunner(config.Request{
		Provider: "p",
		Method:   http.MethodPost,
		Path:     "/body",
		Paginate: true,
		Page: config.PageParams{
			Location:   "body",
			StartField: "startAt",
			LimitField: "maxResults",
			TotalField: "total",
			ReqStart:   "startAt",
			ReqLimit:   "maxResults",
		},
		BodyJSON: map[string]any{"constant": "x"}, // will be retained and merged with pagination
	})
	acc, pages, status, err := r.Do(t.Context())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 3, pages)

	merged := acc["merged"].(map[string]any)
	items := merged["items"]
	switch v := items.(type) {
	case []map[string]any:
		require.Len(t, v, 3)
	case []any:
		require.Len(t, v, 3)
	default:
		t.Fatalf("unexpected items type: %T", v)
	}
}
