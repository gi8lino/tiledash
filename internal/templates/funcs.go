package templates

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// templateAdd returns the sum of two integers.
func templateAdd(a, b int) int {
	return a + b
}

// templateAppend appends a value to a slice and returns the updated slice.
func templateAppend(slice []any, val any) []any {
	return append(slice, val)
}

// templateDict returns a map created from alternating key/value arguments.
// Panics if an odd number of arguments or a non-string key is passed.
func templateDict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict expects even number of args")
	}
	m := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings (got %T)", values[i])
		}
		m[key] = values[i+1]
	}
	return m, nil
}

// templateDig returns the string value of m[key] if it exists and is a string.
// If m is itself a string, it is returned directly.
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

// formatDate parses a Jira timestamp and returns it formatted using the provided layout.
// If parsing fails, the original string is returned.
func formatDate(input, layout string) string {
	input = strings.Replace(input, "Z", "+0000", 1) // normalize timezone
	parsed, err := time.Parse("2006-01-02T15:04:05.000-0700", input)
	if err != nil {
		return input
	}
	return parsed.Format(layout)
}

// templateKeys returns all keys in a map as a string slice.
func templateKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// templateList returns a slice of strings from variadic string arguments.
func templateList(items ...string) []string {
	return items
}

// templateListAny returns a slice of any from variadic arguments.
func templateListAny(items ...any) []any {
	return items
}

// templateSet sets a key-value pair in a map and returns the map.
func templateSet(m map[string]any, key string, val any) map[string]any {
	m[key] = val
	return m
}

// templateSlice returns a slice of any from variadic arguments.
func templateSlice(args ...any) []any {
	return args
}
