package templates

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
)

// TemplateFuncMap returns all helper functions for templates.
func TemplateFuncMap() template.FuncMap {
	fm := sprig.HtmlFuncMap()

	// Custom helpers
	fm["formatJiraDate"] = formatJiraDate
	fm["setany"] = setany
	fm["dig"] = templateDig
	fm["sortBy"] = sortBy
	fm["appendSlice"] = appendSlice
	fm["uniq"] = uniq
	fm["defaultStr"] = defaultStr
	fm["typeOf"] = typeOf
	fm["sumBy"] = sumBy

	return fm
}

// setany sets m[key] = val for map[string]any and returns the map.
func setany(m map[string]any, key string, val any) map[string]any {
	m[key] = val
	return m
}

// appendSlice appends any item to []any; used to build slices in templates.
func appendSlice(slice any, item any) []any {
	switch s := slice.(type) {
	case []any:
		return append(s, item)
	default:
		return []any{item}
	}
}

// templateDig safely accesses string fields in maps or returns the string itself.
func templateDig(m any, key string) string {
	switch v := m.(type) {
	case map[string]any:
		if val, ok := v[key]; ok {
			if s, ok := val.(string); ok {
				return s
			}
		}
	case string:
		return v
	}
	return ""
}

// formatJiraDate parses and formats a Jira timestamp or returns original string.
func formatJiraDate(input, layout string) string {
	input = strings.Replace(input, "Z", "+0000", 1)
	parsed, err := time.Parse("2006-01-02T15:04:05.000-0700", input)
	if err != nil {
		return input
	}
	return parsed.Format(layout)
}

// sortBy returns a sorted copy of a slice of map[string]any,
// sorted by the given field name in ascending or descending order.
func sortBy(field string, desc bool, value any) []any {
	list, ok := value.([]any)
	if !ok {
		panic(fmt.Sprintf("sortBy expects []any, got %T", value))
	}

	sorted := append([]any(nil), list...)

	sort.SliceStable(sorted, func(i, j int) bool {
		mi := sorted[i].(map[string]any)
		mj := sorted[j].(map[string]any)

		vi := mi[field]
		vj := mj[field]

		switch vi := vi.(type) {
		case int:
			return compareFloat64(float64(vi), vj, desc)
		case int64:
			return compareFloat64(float64(vi), vj, desc)
		case float64:
			return compareFloat64(vi, vj, desc)
		case string:
			vjs, _ := vj.(string)
			if desc {
				return vi > vjs
			}
			return vi < vjs
		case time.Time:
			vjt, _ := vj.(time.Time)
			if desc {
				return vi.After(vjt)
			}
			return vi.Before(vjt)
		default:
			return false
		}
	})

	return sorted
}

// compareFloat64 compares vi (already float64) with vj (any numeric), descending if desc is true.
func compareFloat64(vi float64, vj any, desc bool) bool {
	var vjFloat float64
	switch v := vj.(type) {
	case int:
		vjFloat = float64(v)
	case int64:
		vjFloat = float64(v)
	case float64:
		vjFloat = v
	default:
		return false
	}

	if desc {
		return vi > vjFloat
	}
	return vi < vjFloat
}

// uniq returns a list of unique strings from a slice.
func uniq(input []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(input))
	for _, s := range input {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// defaultStr returns fallback if value is empty or whitespace.
func defaultStr(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

// typeOf returns the Go type name of the given value.
func typeOf(v any) string {
	return fmt.Sprintf("%T", v)
}

// sumBy returns the sum of a field in a slice of map[string]any.
func sumBy(field string, items []map[string]any) float64 {
	var total float64
	for _, item := range items {
		switch v := item[field].(type) {
		case int:
			total += float64(v)
		case int64:
			total += float64(v)
		case float64:
			total += v
		}
	}
	return total
}
