package templates

import (
	"html/template"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTemplateFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("contains required helper functions", func(t *testing.T) {
		t.Parallel()

		funcs := TemplateFuncMap()

		assert.Contains(t, funcs, "formatJiraDate")
		assert.Contains(t, funcs, "setany")
		assert.Contains(t, funcs, "dig")

		// Confirm each is a function
		assert.IsType(t, func(string, string) string { return "" }, funcs["formatJiraDate"])
		assert.IsType(t, func(map[string]any, string, any) map[string]any { return nil }, funcs["setany"])
		assert.IsType(t, func(any, string) string { return "" }, funcs["dig"])
	})

	t.Run("includes sprig helpers", func(t *testing.T) {
		t.Parallel()

		funcs := TemplateFuncMap()
		// Check for a well-known sprig function
		assert.Contains(t, funcs, "upper")
		assert.IsType(t, func(string) string { return "" }, funcs["upper"])
	})

	t.Run("can be used to build a valid template", func(t *testing.T) {
		t.Parallel()

		tmpl := template.New("test").Funcs(TemplateFuncMap())
		_, err := tmpl.Parse(`{{define "t"}}{{"hello" | upper}}{{end}}`)
		assert.NoError(t, err)
	})
}

func TestSetAny(t *testing.T) {
	t.Parallel()

	t.Run("sets and returns updated map", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"foo": "bar"}
		out := setany(m, "baz", 123)

		assert.Equal(t, 123, out["baz"])
		assert.Equal(t, "bar", out["foo"])
	})
}

func TestTemplateDig(t *testing.T) {
	t.Parallel()

	t.Run("returns value from map[string]any", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"x": "value"}
		out := templateDig(m, "x")
		assert.Equal(t, "value", out)
	})

	t.Run("returns empty string if key is missing", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"x": "value"}
		out := templateDig(m, "missing")
		assert.Equal(t, "", out)
	})

	t.Run("returns empty string if key is not a string", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"x": 42}
		out := templateDig(m, "x")
		assert.Equal(t, "", out)
	})

	t.Run("returns string directly if input is string", func(t *testing.T) {
		t.Parallel()
		out := templateDig("just a string", "ignored")
		assert.Equal(t, "just a string", out)
	})

	t.Run("returns empty string for unsupported input type", func(t *testing.T) {
		t.Parallel()
		out := templateDig(12345, "ignored")
		assert.Equal(t, "", out)
	})
}

func TestFormatJiraDate(t *testing.T) {
	t.Parallel()

	t.Run("formats valid Jira timestamp", func(t *testing.T) {
		t.Parallel()
		in := "2023-08-01T14:30:00.000Z"
		layout := "2006-01-02 15:04"
		expected := "2023-08-01 14:30"
		out := formatJiraDate(in, layout)
		assert.Equal(t, expected, out)
	})

	t.Run("returns input on invalid timestamp", func(t *testing.T) {
		t.Parallel()
		in := "invalid-date"
		layout := time.RFC822
		out := formatJiraDate(in, layout)
		assert.Equal(t, in, out)
	})
}
