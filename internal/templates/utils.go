package templates

import (
	"html/template"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
)

// TemplateFuncMap returns all helper functions for templates.
func TemplateFuncMap() template.FuncMap {
	fm := sprig.HtmlFuncMap()
	fm["formatJiraDate"] = formatJiraDate
	fm["setany"] = setany
	fm["dig"] = templateDig
	return fm
}

// setany sets m[key] = val for map[string]any and returns the map.
func setany(m map[string]any, key string, val any) map[string]any {
	m[key] = val
	return m
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

// formatJiraDate parses a Jira timestamp and returns it formatted using the provided layout.
// If parsing fails, the original string is returned.
func formatJiraDate(input, layout string) string {
	input = strings.Replace(input, "Z", "+0000", 1) // normalize timezone
	parsed, err := time.Parse("2006-01-02T15:04:05.000-0700", input)
	if err != nil {
		return input
	}
	return parsed.Format(layout)
}
