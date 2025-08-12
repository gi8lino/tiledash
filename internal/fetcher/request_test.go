package fetcher

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestSpec_Normalize(t *testing.T) {
	t.Parallel()

	t.Run("relative URL with sorted query and headers in key", func(t *testing.T) {
		t.Parallel()

		base, _ := url.Parse("https://api.example.com/base/")
		spec := RequestSpec{
			URL:    "../v1/resource",
			Method: "", // should default to GET
			Query: map[string]string{
				"b": "2",
				"a": "1",
				"":  "ignored",
				"x": "", // ignored value
			},
			Headers: http.Header{
				"Z-Last":   []string{"z"},
				"A-First":  []string{"a1", "a2"},
				"Empty":    []string{""}, // included as empty value line
				"Mixed-CA": []string{"v"},
			},
			Body:     []byte("payload"),
			CacheTTL: 0,
		}

		u, key, err := spec.Normalize(base)
		if err != nil {
			t.Fatalf("normalize error: %v", err)
		}
		// URL must be absolute and query sorted as a=1&b=2 (x ignored)
		wantPrefix := "https://api.example.com/v1/resource?a=1&b=2"
		if got := u.String(); got != wantPrefix {
			t.Fatalf("url mismatch:\n got: %s\nwant: %s", got, wantPrefix)
		}
		if key == "" {
			t.Fatalf("empty cache key")
		}
	})
}

func TestCanonicalMethod(t *testing.T) {
	t.Parallel()

	t.Run("defaults to GET", func(t *testing.T) {
		t.Parallel()
		if got := canonicalMethod(""); got != http.MethodGet {
			t.Fatalf("got %q, want GET", got)
		}
	})

	t.Run("uppercases", func(t *testing.T) {
		t.Parallel()
		if got := canonicalMethod("post"); got != http.MethodPost {
			t.Fatalf("got %q, want POST", got)
		}
	})
}

func TestResolveURL(t *testing.T) {
	t.Parallel()

	t.Run("resolves relative against base", func(t *testing.T) {
		t.Parallel()
		base, _ := url.Parse("https://api.example.com/root/")
		u, err := resolveURL(base, "../x/y")
		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com/x/y", u.String())
	})

	t.Run("keeps absolute unchanged", func(t *testing.T) {
		t.Parallel()
		u, err := resolveURL(nil, "http://foo/bar?q=1")
		assert.NoError(t, err)
		assert.Equal(t, "http://foo/bar?q=1", u.String())
	})
}

func TestMergeQuery(t *testing.T) {
	t.Parallel()

	t.Run("deterministic and ignores empty key/value", func(t *testing.T) {
		t.Parallel()
		u, _ := url.Parse("https://api.example.com/x?c=3")
		mergeQuery(u, map[string]string{
			"b": "2",
			"a": "1",
			"c": "override",
			"":  "ignored",
			"d": "",
		})
		// c should be "override"; sorted output order for Encode is stable
		assert.Equal(t, "a=1&b=2&c=override", u.RawQuery)
	})
}

func TestBuildCacheKey(t *testing.T) {
	t.Parallel()

	t.Run("Build same individual cache key each time", func(t *testing.T) {
		t.Parallel()
		u1, _ := url.Parse("https://api.example.com/x?a=1&b=2")
		u2, _ := url.Parse("https://api.example.com/x?b=2&a=1") // same after sort

		h1 := http.Header{
			"X-Foo":  []string{"A"},
			"a-head": []string{"1", "2"},
		}
		h2 := http.Header{
			"A-Head": []string{"1", "2"}, // different case
			"x-foo":  []string{"A"},      // different order
		}

		k1 := buildCacheKey("GET", u1, h1, []byte("payload"))
		k2 := buildCacheKey("GET", u2, h2, []byte("payload"))
		assert.NotEqual(t, k1, k2)

		assert.Equal(t, "026631b5cf5f178f", k1)
		assert.Equal(t, "f65c8e11ad3a06c5", k2)
	})
}
