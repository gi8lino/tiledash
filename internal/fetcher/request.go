package fetcher

import (
	"context"
	"encoding/hex"
	"hash/fnv"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// RequestSpec describes a single HTTP request the transport should execute.
type RequestSpec struct {
	URL      string
	Method   string
	Query    map[string]string
	Headers  http.Header
	Body     []byte
	CacheTTL time.Duration
}

// Normalize resolves base+path, merges query deterministically, and returns a stable cache key.
// It returns the absolute URL to request and the cache key derived from method+URL+headers+body.
func (r *RequestSpec) Normalize(base *url.URL) (u *url.URL, key string, err error) {
	// canonicalize method once
	method := canonicalMethod(r.Method)

	// resolve URL (absolute) against base
	u, err = resolveURL(base, r.URL)
	if err != nil {
		return nil, "", err
	}

	// merge query params in deterministic order (ignoring empty keys/values)
	mergeQuery(u, r.Query)

	// compute stable cache key from method+URL+headers+body
	key = buildCacheKey(method, u, r.Headers, r.Body)
	return u, key, nil
}

// ContextKey is a private type for context values in this package.
type ContextKey string

// IsNoCache reports whether cache should be bypassed.
func IsNoCache(ctx context.Context) bool {
	v, _ := ctx.Value(ContextKey("nocache")).(bool)
	return v
}

// canonicalMethod returns an upper-cased HTTP method or GET if empty.
func canonicalMethod(m string) string {
	// default to GET if unset; otherwise uppercase
	m = strings.TrimSpace(m) // trim whitespace just in case
	if m == "" {
		return http.MethodGet
	}
	return strings.ToUpper(m)
}

// resolveURL parses raw and resolves it against base if not absolute.
// It returns a new *url.URL that is absolute.
func resolveURL(base *url.URL, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if !u.IsAbs() && base != nil {
		u = base.ResolveReference(u)
	}
	return u, nil
}

// mergeQuery merges kv into u.Query() in a deterministic way and updates u.RawQuery.
// Empty keys are ignored; empty values are skipped to avoid surprising "?k=" entries.
func mergeQuery(u *url.URL, kv map[string]string) {
	if u == nil || len(kv) == 0 {
		return
	}
	q := u.Query()

	// collect keys to sort for deterministic order
	keys := make([]string, 0, len(kv))
	for k := range kv {
		if k != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// set each non-empty value
	for _, k := range keys {
		if v := kv[k]; v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
}

// buildCacheKey returns a stable hex key for method+URL+headers+body using FNV-1a 64.
func buildCacheKey(method string, u *url.URL, hdr http.Header, body []byte) string {
	h := fnv.New64a()

	// method + URL line-separated; URL must already include merged query
	h.Write([]byte(method))     // nolint:errcheck
	h.Write([]byte("\n"))       // nolint:errcheck
	h.Write([]byte(u.String())) // nolint:errcheck
	h.Write([]byte("\n"))       // nolint:errcheck

	// headers in deterministic order: lowercased key; value is comma-joined
	if hdr != nil {
		// collect and sort header keys
		keys := make([]string, 0, len(hdr))
		for k := range hdr {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			lk := strings.ToLower(k)
			val := strings.Join(hdr.Values(k), ",")
			h.Write([]byte(lk))   // nolint:errcheck
			h.Write([]byte(":"))  // nolint:errcheck
			h.Write([]byte(val))  // nolint:errcheck
			h.Write([]byte("\n")) // nolint:errcheck
		}
	}

	// include body bytes if any
	if len(body) > 0 {
		h.Write(body) // nolint:errcheck
	}

	return hex.EncodeToString(h.Sum(nil))
}
