package providers

import (
	"encoding/json"
	"maps"
	"strconv"
	"strings"

	"github.com/gi8lino/tiledash/internal/config"
)

// Accumulator is the shape returned to templates (merged + all pages).
type Accumulator map[string]any

// newAccumulator creates an empty accumulator with "merged" and "pages".
func newAccumulator() Accumulator {
	return Accumulator{
		"merged": map[string]any{},                 // per-key concatenated arrays across pages
		"pages":  []map[string]any{},               // raw pages in arrival order
		"__seen": map[string]map[string]struct{}{}, // internal per-key dedupe set
	}
}

// appendPage adds a page payload into acc["pages"].
func appendPage(acc Accumulator, page map[string]any) {
	pages, _ := acc["pages"].([]map[string]any)
	acc["pages"] = append(pages, page)
}

// mergeCommonArrays appends any top-level JSON arrays from page into acc["merged"] by the same key, de-duplicated.
func mergeCommonArrays(acc Accumulator, page map[string]any) {
	merged, _ := acc["merged"].(map[string]any)
	if merged == nil {
		merged = map[string]any{}
		acc["merged"] = merged
	}
	seenAll, _ := acc["__seen"].(map[string]map[string]struct{})
	if seenAll == nil {
		seenAll = map[string]map[string]struct{}{}
		acc["__seen"] = seenAll
	}

	for k, v := range page {
		arr, ok := v.([]any)
		if !ok || len(arr) == 0 {
			continue // only concatenate arrays
		}
		seen, ok := seenAll[k]
		if !ok {
			seen = map[string]struct{}{}
			seenAll[k] = seen
		}
		dst, _ := merged[k].([]any)
		for _, elem := range arr {
			id := itemIdentity(elem)
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			dst = append(dst, elem)
		}
		merged[k] = dst
	}
}

// mergeCommonArraysAndCount merges like mergeCommonArrays and returns how many new items were added across all keys.
func mergeCommonArraysAndCount(acc Accumulator, page map[string]any) int {
	before := totalSeen(acc)
	mergeCommonArrays(acc, page)
	after := totalSeen(acc)
	if after < before {
		return 0 // shouldn't happen, but stay safe
	}
	return after - before
}

// totalSeen returns the total number of unique items tracked in the internal __seen sets.
func totalSeen(acc Accumulator) int {
	seenAll, _ := acc["__seen"].(map[string]map[string]struct{})
	if seenAll == nil {
		return 0
	}
	n := 0
	for _, s := range seenAll {
		n += len(s)
	}
	return n
}

// nextPageParams computes pagination for the next request and whether to continue.
func nextPageParams(cfg config.PageParams, last map[string]any, seenPages int) (nextStart int, nextLimit int, ok bool) {
	// Enforce user-defined page cap first.
	if cfg.LimitPages > 0 && seenPages >= cfg.LimitPages {
		return 0, 0, false
	}

	// Read counters from the last response, tolerating missing or negative values.
	start := asInt(last[cfg.StartField])
	limit := asInt(last[cfg.LimitField])
	total := asInt(last[cfg.TotalField])

	// If the API didn't return limit, fall back to reqLimit name if present (read from response).
	if limit == 0 && strings.TrimSpace(cfg.ReqLimit) != "" {
		if alt := asInt(last[cfg.ReqLimit]); alt > 0 {
			limit = alt
		}
	}
	if limit <= 0 {
		limit = 1 // ensure progress even with bad counters
	}

	// Compute the next window start.
	next := start + limit

	// If total is known and we've reached or passed it, stop.
	if total > 0 && next >= total {
		return 0, 0, false
	}

	return next, limit, true
}

// injectQueryPage returns a copy of q with pagination params injected.
func injectQueryPage(q map[string]string, reqStart, reqLimit string, nextStart, nextLimit int) map[string]string {
	out := map[string]string{}
	maps.Copy(out, q)
	if strings.TrimSpace(reqStart) != "" {
		out[reqStart] = strconv.Itoa(nextStart)
	}
	if strings.TrimSpace(reqLimit) != "" && nextLimit > 0 {
		out[reqLimit] = strconv.Itoa(nextLimit)
	}
	return out
}

// injectBodyPage returns JSON body bytes with pagination fields merged into base.
func injectBodyPage(base map[string]any, reqStart, reqLimit string, nextStart, nextLimit int) []byte {
	m := map[string]any{}
	maps.Copy(m, base)
	if strings.TrimSpace(reqStart) != "" {
		m[reqStart] = nextStart
	}
	if strings.TrimSpace(reqLimit) != "" && nextLimit > 0 {
		m[reqLimit] = nextLimit
	}
	raw, _ := json.Marshal(m) // best-effort marshalling
	return raw
}

// itemIdentity extracts a best-effort identity string for de-duplication.
func itemIdentity(v any) string {
	if m, ok := v.(map[string]any); ok {
		if id, ok := m["id"]; ok {
			return stringify(id)
		}
		if key, ok := m["key"]; ok {
			return stringify(key)
		}
	}
	b, _ := json.Marshal(v) // structural fallback
	return string(b)
}
