package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/containeroo/tinyflags"
	"gopkg.in/yaml.v3"
)

// Config is the mock server configuration root.
type Config struct {
	Port        int     `yaml:"port"`
	DataDir     string  `yaml:"dataDir"`
	RandomDelay bool    `yaml:"randomDelay"`
	Routes      []Route `yaml:"routes"`
}

// Route defines a single HTTP path and how to serve JSON for it.
type Route struct {
	Path       string    `yaml:"path"`                 // e.g. /rest/api/2/search
	ItemsField string    `yaml:"itemsField,omitempty"` // e.g. "issues" | "items" | "data" ; if empty and file is a JSON array, that array is used
	Select     *Select   `yaml:"select,omitempty"`     // how to pick the data file token
	Paginate   *Paginate `yaml:"paginate,omitempty"`   // nil = no pagination
}

// Select configures how the mock chooses which data file to serve.
type Select struct {
	From         string `yaml:"from,omitempty"`         // "query" | "body" | "header" | "static"
	Key          string `yaml:"key,omitempty"`          // name of param/field/header (unused for static)
	Regex        string `yaml:"regex,omitempty"`        // optional regex with 1 capture group used as token
	FileTemplate string `yaml:"fileTemplate,omitempty"` // template like "%s.json" (default)
	Static       string `yaml:"static,omitempty"`       // used when From == "static"
}

// Paginate defines pagination mechanics for a route.
type Paginate struct {
	Location     string `yaml:"location"`               // "query" | "body"
	StartField   string `yaml:"startField"`             // e.g. "startAt"
	LimitField   string `yaml:"limitField"`             // e.g. "maxResults"
	TotalField   string `yaml:"totalField"`             // e.g. "total"
	ReqStart     string `yaml:"reqStart"`               // e.g. "startAt" (request param/body key)
	ReqLimit     string `yaml:"reqLimit"`               // e.g. "maxResults" (request param/body key)
	DefaultStart int    `yaml:"defaultStart,omitempty"` // default if absent (default 0)
	DefaultLimit int    `yaml:"defaultLimit,omitempty"` // default if absent (default 50)
}

// main starts the mock server with required YAML config and per-route selection.
func main() {
	var (
		flagConfigPath string
		flagLogBody    bool
	)

	tf := tinyflags.NewFlagSet("mock-server", tinyflags.ExitOnError)
	tf.StringVar(&flagConfigPath, "config", "", "Path to mock-server config.yaml (required)").Value()
	tf.BoolVar(&flagLogBody, "log-body", false, "Log JSON request bodies (may contain secrets)")

	if err := tf.Parse(os.Args[1:]); err != nil {
		log.Fatal("flag parse error:", err)
	}

	if strings.TrimSpace(flagConfigPath) == "" {
		log.Fatal("missing required --config=<path to yaml>")
	}

	cfg, err := loadConfig(flagConfigPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// absolute stays absolute
	if !filepath.IsAbs(cfg.DataDir) {
		base := filepath.Dir(flagConfigPath)
		cfg.DataDir, _ = filepath.Abs(filepath.Join(base, cfg.DataDir))
	}

	mux := http.NewServeMux()
	for i := range cfg.Routes {
		rt := cfg.Routes[i] // capture per-iteration
		validateRouteOrDie(rt)
		mux.HandleFunc(rt.Path, func(w http.ResponseWriter, r *http.Request) {
			if cfg.RandomDelay {
				applyRandomDelay(200, 1000)
			}
			logRequest(r, flagLogBody)
			handleRoute(w, r, cfg, rt)
		})
		log.Printf("route mounted: %s", rt.Path)
	}

	addr := ":" + strconv.Itoa(cfg.Port)
	log.Printf("Mock server listening on %s (data-dir: %s)", addr, cfg.DataDir)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// loadConfig reads and validates the YAML configuration file.
func loadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, err
	}

	// Basic defaults.
	if cfg.Port == 0 {
		cfg.Port = 8081
	}
	if strings.TrimSpace(cfg.DataDir) == "" {
		cfg.DataDir = "./data"
	}

	// Select defaults per route.
	for i := range cfg.Routes {
		rt := &cfg.Routes[i]
		if rt.Select == nil {
			return Config{}, fmt.Errorf("route %q: missing select block", rt.Path)
		}
		if rt.Select.FileTemplate == "" {
			rt.Select.FileTemplate = "%s.json"
		}
		// Paginate defaults when present.
		if p := rt.Paginate; p != nil {
			if p.DefaultLimit <= 0 {
				p.DefaultLimit = 50
			}
			// DefaultStart naturally zero.
		}
	}

	return cfg, nil
}

// validateRouteOrDie ensures minimal correctness of a single route.
func validateRouteOrDie(rt Route) {
	if strings.TrimSpace(rt.Path) == "" {
		log.Fatalf("invalid route: empty path")
	}
	if rt.Select == nil {
		log.Fatalf("route %s: select is required", rt.Path)
	}
	if strings.EqualFold(rt.Select.From, "static") && strings.TrimSpace(rt.Select.Static) == "" {
		log.Fatalf("route %s: select.static must be set when select.from=static", rt.Path)
	}
}

// handleRoute processes one request for a configured route.
func handleRoute(w http.ResponseWriter, r *http.Request, cfg Config, rt Route) {
	token, err := selectDataToken(r, rt.Select)
	if err != nil {
		http.Error(w, "selection error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Determine file name.
	var fileName string
	if strings.EqualFold(rt.Select.From, "static") {
		fileName = rt.Select.Static
	} else {
		fileName = fmt.Sprintf(rt.Select.FileTemplate, token)
	}

	// Read JSON payload file.
	filePath := filepath.Join(cfg.DataDir, fileName)
	raw, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "mock data not found: "+filePath, http.StatusNotFound)
		return
	}

	// No pagination → write as-is.
	if rt.Paginate == nil {
		writeJSON(w, http.StatusOK, raw)
		return
	}

	// Decode to find the array to paginate.
	payload, items, isArray, err := decodeForPagination(raw, rt.ItemsField)
	if err != nil {
		http.Error(w, "invalid mock JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Resolve start/limit from query/body according to route config.
	start, limit := resolveReqPaging(r, *rt.Paginate)

	// Build page envelope with counters and sliced items.
	page, err := buildPaginatedPage(payload, items, isArray, *rt.Paginate, start, limit)
	if err != nil {
		http.Error(w, "paginate error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode and send.
	b, _ := json.Marshal(page)
	writeJSON(w, http.StatusOK, b)
}

// selectDataToken extracts the file token according to the route's Select config.
func selectDataToken(r *http.Request, s *Select) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s.From)) {
	case "static":
		return s.Static, nil

	case "query":
		val := r.URL.Query().Get(s.Key)
		return applyRegex(val, s.Regex)

	case "header":
		val := r.Header.Get(s.Key)
		return applyRegex(val, s.Regex)

	case "body":
		if r.Body == nil {
			return "", errors.New("empty body")
		}
		b, _ := io.ReadAll(r.Body)
		defer func() { r.Body = io.NopCloser(strings.NewReader(string(b))) }()
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return "", fmt.Errorf("invalid JSON body: %w", err)
		}
		raw, _ := m[s.Key].(string)
		return applyRegex(raw, s.Regex)

	default:
		return "", fmt.Errorf("unsupported select.from=%q", s.From)
	}
}

// applyRegex returns the first capture group if regex is provided, otherwise the raw value.
func applyRegex(s, re string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("empty selection value")
	}
	if strings.TrimSpace(re) == "" {
		return s, nil
	}
	rx, err := regexp.Compile(re)
	if err != nil {
		return "", fmt.Errorf("bad regex: %w", err)
	}
	m := rx.FindStringSubmatch(s)
	if len(m) < 2 {
		return "", errors.New("no regex capture match")
	}
	return m[1], nil
}

// decodeForPagination parses JSON and returns payload, pointer to items slice, and whether top-level was an array.
func decodeForPagination(raw []byte, itemsField string) (payload any, items *[]any, isArray bool, err error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err = dec.Decode(&payload); err != nil {
		return nil, nil, false, err
	}

	// Top-level array case.
	if arr, ok := payload.([]any); ok {
		isArray = true
		items = &arr
		return payload, items, isArray, nil
	}

	// Top-level object case.
	if obj, ok := payload.(map[string]any); ok {
		// If itemsField is set, use it; else try common defaults.
		if strings.TrimSpace(itemsField) == "" {
			for _, k := range []string{"issues", "items", "data"} {
				if v, ok := obj[k]; ok {
					if arr, ok := v.([]any); ok {
						return payload, &arr, false, nil
					}
				}
			}
			return payload, nil, false, errors.New("itemsField not found; set route.itemsField")
		}
		v, ok := obj[itemsField]
		if !ok {
			return payload, nil, false, fmt.Errorf("itemsField %q not present", itemsField)
		}
		arr, ok := v.([]any)
		if !ok {
			return payload, nil, false, fmt.Errorf("itemsField %q is not an array", itemsField)
		}
		return payload, &arr, false, nil
	}

	return payload, nil, false, errors.New("unsupported JSON shape")
}

// resolveReqPaging extracts start/limit from query/body according to paginate config.
func resolveReqPaging(r *http.Request, p Paginate) (start, limit int) {
	// Defaults.
	if p.DefaultLimit <= 0 {
		p.DefaultLimit = 50
	}
	start = p.DefaultStart
	limit = p.DefaultLimit

	// Query-based pagination.
	if strings.EqualFold(p.Location, "query") {
		if v := r.URL.Query().Get(p.ReqStart); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				start = n
			}
		}
		if v := r.URL.Query().Get(p.ReqLimit); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		return
	}

	// Body-based pagination.
	if strings.EqualFold(p.Location, "body") {
		if r.Body != nil {
			defer r.Body.Close() // best-effort
			b, _ := io.ReadAll(r.Body)
			if len(b) > 0 {
				var m map[string]any
				if json.Unmarshal(b, &m) == nil {
					if v, ok := m[p.ReqStart]; ok {
						if n := asInt(v); n >= 0 {
							start = n
						}
					}
					if v, ok := m[p.ReqLimit]; ok {
						if n := asInt(v); n > 0 {
							limit = n
						}
					}
				}
				// restore body for any downstream reads
				r.Body = io.NopCloser(strings.NewReader(string(b)))
			}
		}
	}
	return
}

// buildPaginatedPage produces a response with start/limit/total injected and items sliced.
func buildPaginatedPage(payload any, items *[]any, isArray bool, p Paginate, start, limit int) (map[string]any, error) {
	// No items to paginate → wrap payload with counters but no items slicing.
	if items == nil {
		return map[string]any{
			p.StartField: start,
			p.LimitField: max(limit, 1),
			p.TotalField: 0,
			"payload":    payload,
		}, nil
	}

	total := len(*items)

	// Clamp and ensure progress.
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	if limit <= 0 {
		limit = 1
	}

	end := start + limit
	if end > total {
		end = total
	}
	if start > end {
		start = end
	}

	pageSlice := (*items)[start:end]

	// Envelope counters common to both shapes.
	resp := map[string]any{
		p.StartField: start,
		p.LimitField: limit,
		p.TotalField: total,
	}

	// Top-level array → synthesize "items".
	if isArray {
		resp["items"] = pageSlice
		return resp, nil
	}

	// Top-level object → clone and replace items field.
	obj, _ := payload.(map[string]any)
	out := make(map[string]any, len(obj)+3)
	maps.Copy(out, obj)

	itemsField := detectItemsField(out)
	if itemsField == "" {
		return nil, errors.New("cannot determine items field to replace")
	}

	out[itemsField] = pageSlice
	out[p.StartField] = start
	out[p.LimitField] = limit
	out[p.TotalField] = total
	return out, nil
}

// detectItemsField returns the first known items field present.
func detectItemsField(m map[string]any) string {
	for _, k := range []string{"issues", "items", "data"} {
		if v, ok := m[k]; ok {
			if _, ok := v.([]any); ok {
				return k
			}
		}
	}
	return ""
}

// writeJSON writes a JSON response with status and bytes.
func writeJSON(w http.ResponseWriter, status int, raw []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

// applyRandomDelay sleeps for a random duration between minMs and maxMs.
func applyRandomDelay(minMs, maxMs int) {
	if maxMs <= minMs {
		maxMs = minMs + 1
	}
	delta := rand.Intn(maxMs-minMs) + minMs
	time.Sleep(time.Duration(delta) * time.Millisecond)
}

// logRequest logs method, path, query, headers and optionally the JSON body.
func logRequest(r *http.Request, logBody bool) {
	redacted := http.Header{}
	for k, vv := range r.Header {
		if strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "Cookie") {
			redacted[k] = []string{"<redacted>"}
		} else {
			redacted[k] = vv
		}
	}

	var bodyPreview string
	if logBody && r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		bodyPreview = string(b)
		r.Body = io.NopCloser(strings.NewReader(bodyPreview))
	}

	log.Printf("REQ %s %s?%s headers=%v body=%s",
		r.Method, r.URL.Path, r.URL.RawQuery, redacted, truncate(bodyPreview, 2048))
}

// asInt converts numeric JSON values to int safely.
func asInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		if n < 0 {
			return 0
		}
		return int(n)
	case string:
		n, _ := strconv.Atoi(t)
		if n < 0 {
			return 0
		}
		return n
	default:
		return 0
	}
}

// truncate returns at most n runes of s.
func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// max returns the larger of a and b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
