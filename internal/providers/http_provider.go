package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gi8lino/tiledash/internal/cache"
	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/fetcher"
)

// HTTPProvider represents a single configured upstream (baseURL + auth + client).
type HTTPProvider struct {
	Name   string
	Base   *url.URL
	Auth   *config.AuthConfig
	Client *http.Client
	Cache  *cache.MemCache
}

// NewHTTPProvider constructs an HTTPProvider from config.
func NewHTTPProvider(name string, pc config.Provider) (*HTTPProvider, error) {
	if strings.TrimSpace(pc.BaseURL) == "" {
		return nil, fmt.Errorf("provider %q: missing baseURL", name)
	}
	u, err := url.Parse(pc.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("provider %q: invalid baseURL: %w", name, err)
	}
	return &HTTPProvider{
		Name:   name,
		Base:   u,
		Auth:   &pc.Auth,
		Client: newHTTPClient(pc),
		Cache:  cache.NewMemCache(),
	}, nil
}

// HTTPRunner executes one request (with optional pagination) against a provider.
type HTTPRunner struct {
	prov *HTTPProvider
	req  config.Request

	// Invariants we canonicalize once to avoid recomputing on every call.
	method      string      // upper-cased HTTP method, defaulting to GET
	baseHeaders http.Header // canonical headers derived from req.Headers
	baseBody    map[string]any

	// Pre-normalized data for the non-paginated fast path.
	// When req.Paginate == false, these are filled and reused on every Do().
	preURL      *url.URL // absolute, normalized URL with merged query
	preCacheKey string   // stable hash from Normalize(method+URL+headers+body)
	preBody     []byte   // exact body bytes we will send (nil for no body)
	preTTL      time.Duration
	preHeaders  http.Header // includes content-type if needed
}

// NewRunner prepares a runnable request bound to this provider.
func (p *HTTPProvider) NewRunner(req config.Request) *HTTPRunner {
	// Canonicalize method once.
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	// Canonicalize headers once (request headers are map[string]string at config level).
	hdr := http.Header{}
	for k, v := range req.Headers {
		if k != "" && v != "" {
			hdr.Set(k, v)
		}
	}

	// Seed base body map for pagination/body merging; copy once.
	baseBody := map[string]any{}
	if len(req.BodyJSON) > 0 {
		maps.Copy(baseBody, req.BodyJSON)
	}

	r := &HTTPRunner{
		prov:        p,
		req:         req,
		method:      method,
		baseHeaders: hdr,
		baseBody:    baseBody,
	}

	// If the request is not paginated, fully normalize once and cache fields for reuse.
	if !req.Paginate {
		// Build the exact body we will send.
		var bodyBytes []byte
		var contentType string
		switch {
		case len(baseBody) > 0 && strings.TrimSpace(req.Body) == "":
			raw, _ := json.Marshal(baseBody)
			bodyBytes = raw
			contentType = "application/json"
		case strings.TrimSpace(req.Body) != "":
			bodyBytes = []byte(req.Body)
		default:
			bodyBytes = nil
		}

		// Ensure content-type if we built a JSON body and caller didn't set it.
		hdrCopy := hdr.Clone()
		if contentType != "" && hdrCopy.Get("Content-Type") == "" {
			hdrCopy.Set("Content-Type", contentType)
		}

		spec := fetcher.RequestSpec{
			URL:      req.Path,
			Method:   method,
			Query:    req.Query,
			Headers:  hdrCopy,
			Body:     bodyBytes,
			CacheTTL: req.TTL,
		}

		u, key, err := spec.Normalize(p.Base)
		if err == nil {
			// Only set the fast-path fields if normalization succeeded.
			r.preURL = u
			r.preCacheKey = key
			r.preBody = bodyBytes
			r.preTTL = req.TTL
			r.preHeaders = hdrCopy
		}
		// If normalization failed, we’ll just fall back to the paginated path’s per-call normalization.
	}

	return r
}

// Do executes the configured request (paginated or not) and returns accumulator, pageCount, and HTTP status.
func (r *HTTPRunner) Do(ctx context.Context) (Accumulator, int, int, error) {
	if !r.req.Paginate {
		return r.runNonPaginated(ctx)
	}
	return r.runPaginated(ctx)
}

// runNonPaginated executes a single non-paginated request, using pre-normalized fields when available.
func (r *HTTPRunner) runNonPaginated(ctx context.Context) (Accumulator, int, int, error) {
	// Fast path: use pre-normalized URL/cache key/body/headers if NewRunner succeeded in precomputing them.
	if r.preURL != nil {
		useCache := r.preTTL > 0 && !fetcher.IsNoCache(ctx)
		if useCache {
			if page, ok := r.prov.Cache.Get(r.preCacheKey); ok {
				acc := newAccumulator()
				appendPage(acc, page)
				mergeCommonArrays(acc, page)
				return acc, 1, http.StatusOK, nil
			}
		}

		// Build request with precomputed body bytes (if any).
		var body io.Reader
		if len(r.preBody) > 0 {
			body = bytes.NewReader(r.preBody)
		}

		req, err := http.NewRequestWithContext(ctx, r.method, r.preURL.String(), body)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("build request: %w", err)
		}
		req.Header = r.preHeaders.Clone() // don't mutate cached headers
		applyAuth(req, r.prov.Auth)

		res, err := r.prov.Client.Do(req)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("request failed: %w", err)
		}
		defer res.Body.Close() // nolint:errcheck

		raw, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, 0, res.StatusCode, fmt.Errorf("read body: %w", err)
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, 0, res.StatusCode, fmt.Errorf("upstream %d: %s", res.StatusCode, string(trim(raw, 2048)))
		}

		var page map[string]any
		if len(raw) == 0 {
			page = map[string]any{}
		} else if err := json.Unmarshal(raw, &page); err != nil {
			return nil, 0, res.StatusCode, fmt.Errorf("invalid JSON: %w", err)
		}

		if useCache {
			r.prov.Cache.Set(r.preCacheKey, page, r.preTTL)
		}

		acc := newAccumulator()
		appendPage(acc, page)
		mergeCommonArrays(acc, page)
		return acc, 1, res.StatusCode, nil
	}

	// Fallback: normalize once on the fly (e.g., if precompute failed).
	status, page, err := r.doOnceNormalized(ctx, r.method, r.req.Path, r.req.Query, r.baseHeaders, nil, r.baseBody, r.req.TTL)
	if err != nil {
		return nil, 0, status, err
	}
	acc := newAccumulator()
	appendPage(acc, page)
	mergeCommonArrays(acc, page)
	return acc, 1, status, nil
}

// runPaginated executes the paginated request loop, normalizing each page while reusing invariants.
func (r *HTTPRunner) runPaginated(ctx context.Context) (acc Accumulator, pages int, status int, err error) {
	acc = newAccumulator()
	pageCount := 0
	prevStart := -1 // sentinel to detect no-progress loops

	// Start with the base query; body pagination will replace this with JSON body bytes.
	nextQ := r.req.Query
	var nextBodyRaw []byte

	for {
		// Execute one page.
		status, page, err := r.doOnceNormalized(ctx, r.method, r.req.Path, nextQ, r.baseHeaders, nextBodyRaw, r.baseBody, r.req.TTL)
		if err != nil {
			return acc, pageCount, status, err
		}
		appendPage(acc, page)

		// Merge and count how many **new** items we actually added to the accumulator.
		added := mergeCommonArraysAndCount(acc, page)
		pageCount++

		// Detect "no progress" loops:
		// - backend ignores start/limit and returns the same window,
		// - or windows overlap fully after de-dup.
		curStart := asInt(page[r.req.Page.StartField])
		if added == 0 || (prevStart >= 0 && curStart == prevStart) {
			// graceful stop: we reached the end or the backend isn't advancing
			return acc, pageCount, status, nil
		}
		prevStart = curStart

		// Compute next request parameters (stop if total indicates completion).
		ns, nl, ok := nextPageParams(r.req.Page, page, pageCount)
		if !ok {
			return acc, pageCount, status, nil
		}

		// Decide where pagination params live and prepare the next request.
		switch strings.ToUpper(strings.TrimSpace(r.req.Page.Location)) {
		case "BODY":
			nextBodyRaw = injectBodyPage(r.baseBody, r.req.Page.ReqStart, r.req.Page.ReqLimit, ns, nl)
			nextQ = r.req.Query // keep base query stable when paginating in body
		default:
			nextQ = injectQueryPage(r.req.Query, r.req.Page.ReqStart, r.req.Page.ReqLimit, ns, nl)
			nextBodyRaw = nil
		}
	}
}

// doOnceNormalized normalizes one request, executes it with auth/cache, and decodes JSON.
func (r *HTTPRunner) doOnceNormalized(
	ctx context.Context,
	method string,
	path string,
	query map[string]string,
	headers http.Header,
	bodyRaw []byte,
	bodyJSON map[string]any,
	ttl time.Duration,
) (status int, page map[string]any, err error) {
	// Build exact body bytes and content type once; these bytes also feed the cache key.
	var (
		body        io.Reader
		contentType string
		bodyBytes   []byte
	)
	switch {
	case len(bodyJSON) > 0 && len(bytes.TrimSpace(bodyRaw)) == 0:
		raw, _ := json.Marshal(bodyJSON) // best-effort; upstream may still reject
		bodyBytes = raw
		body = bytes.NewReader(raw)
		contentType = "application/json"
	case len(bytes.TrimSpace(bodyRaw)) > 0:
		// Raw payload provided (e.g., already-encoded JSON or other types).
		bodyBytes = bodyRaw
		body = bytes.NewReader(bodyBytes)
	default:
		// No body at all.
		bodyBytes = nil
		body = nil
	}

	// Clone headers and ensure Content-Type if we constructed a JSON body.
	hdr := headers.Clone()
	if contentType != "" && hdr.Get("Content-Type") == "" {
		hdr.Set("Content-Type", contentType)
	}

	// Normalize: absolute URL + deterministic cache key.
	spec := fetcher.RequestSpec{
		URL:      path,
		Method:   method,
		Query:    query,
		Headers:  hdr,
		Body:     bodyBytes,
		CacheTTL: ttl,
	}
	u, cacheKey, nerr := spec.Normalize(r.prov.Base)
	if nerr != nil {
		return status, page, fmt.Errorf("normalize request: %w", nerr)
	}

	// Cache lookup if allowed.
	useCache := ttl > 0 && !fetcher.IsNoCache(ctx)
	if useCache {
		if cached, ok := r.prov.Cache.Get(cacheKey); ok {
			return http.StatusOK, cached, nil
		}
	}

	// Execute HTTP request.
	req, berr := http.NewRequestWithContext(ctx, method, u.String(), body)
	if berr != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("build request: %w", berr)
	}

	req.Header = hdr
	applyAuth(req, r.prov.Auth)

	res, rerr := r.prov.Client.Do(req)
	if rerr != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("request failed: %w", rerr)
	}
	defer res.Body.Close() // nolint:errcheck

	raw, rderr := io.ReadAll(res.Body)
	status = res.StatusCode
	if rderr != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("read body: %w", rderr)
	}
	if status < 200 || status >= 300 {
		return http.StatusInternalServerError, nil, fmt.Errorf("upstream %d: %s", status, string(trim(raw, 2048)))
	}

	// Decode JSON with UseNumber.
	page, err = decodeJSONUseNumber(raw)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Store in cache if enabled.
	if useCache {
		r.prov.Cache.Set(cacheKey, page, ttl)
	}
	return
}

// decodeJSONUseNumber decodes JSON into a map using UseNumber to preserve integer precision.
func decodeJSONUseNumber(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
