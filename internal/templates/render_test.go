package templates

import (
	"encoding/json"
	"html/template"
	"reflect"
	"testing"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderError(t *testing.T) {
	t.Parallel()

	t.Run("Error message", func(t *testing.T) {
		t.Parallel()
		err := NewRenderError("test-type", "test-message", "test-detail")
		assert.Equal(t, "test-type: test-message (test-detail)", err.Error())
	})
}

func TestRenderCell(t *testing.T) {
	t.Parallel()

	t.Run("renders valid tile with map data", func(t *testing.T) {
		t.Parallel()

		// Access map keys via the built-in "index" func
		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}<div>{{.Title}}: {{ index .Data "key" }}</div>{{end}}`))

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{
				{
					Title:    "Test",
					Template: "s1",
				},
			},
		}

		html, rerr := RenderCell(t.Context(), 0, cfg, tmpl, map[string]any{"key": "value"})
		require.Nil(t, rerr)
		assert.Equal(t, "<div>Test: value</div>", string(html))
	})

	t.Run("renders valid tile with []byte JSON data", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}<span>{{ index .Data "k" }}</span>{{end}}`))

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{{Title: "T", Template: "s1"}},
		}

		html, rerr := RenderCell(t.Context(), 0, cfg, tmpl, []byte(`{"k":"v"}`))
		assert.Nil(t, rerr)
		assert.Equal(t, "<span>v</span>", string(html))
	})

	t.Run("uses merged when accumulator has merged + pages", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}{{ len (index .Data "issues") }}{{end}}`))

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{{Title: "A", Template: "s1"}},
		}

		acc := map[string]any{
			"merged": map[string]any{
				"issues": []any{map[string]any{"id": 1}, map[string]any{"id": 2}},
			},
			"pages": []map[string]any{
				{"issues": []any{map[string]any{"id": 1}}},
				{"issues": []any{map[string]any{"id": 2}}},
			},
		}

		html, rerr := RenderCell(t.Context(), 0, cfg, tmpl, acc)
		require.Nil(t, rerr)
		assert.Equal(t, "2", string(html))
	})

	t.Run("falls back to first page when no merged", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}{{ index .Data "first" }}{{end}}`))

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{{Title: "A", Template: "s1"}},
		}

		acc := map[string]any{
			"pages": []map[string]any{
				{"first": "yes"},
				{"first": "no"},
			},
		}

		html, rerr := RenderCell(t.Context(), 0, cfg, tmpl, acc)
		require.Nil(t, rerr)
		assert.Equal(t, "yes", string(html))
	})

	t.Run("supports named map types via reflection", func(t *testing.T) {
		t.Parallel()

		type Acc map[string]any
		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}{{ index .Data "x" }}{{end}}`))
		cfg := config.DashboardConfig{
			Tiles: []config.Tile{{Title: "N", Template: "s1"}},
		}

		html, rerr := RenderCell(t.Context(), 0, cfg, tmpl, Acc{"x": "ok"})
		require.Nil(t, rerr)
		assert.Equal(t, "ok", string(html))
	})

	t.Run("handles JSON parsing error for invalid []byte", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}ok{{end}}`))
		cfg := config.DashboardConfig{
			Tiles: []config.Tile{{Title: "Broken JSON", Template: "s1"}},
		}

		html, rerr := RenderCell(t.Context(), 0, cfg, tmpl, []byte(`{invalid json}`))
		require.Error(t, rerr)
		assert.EqualError(t, rerr, "json: Response could not be parsed (invalid character 'i' looking for beginning of object key string)")
		assert.Empty(t, html)
	})

	t.Run("handles template render error (missing template name)", func(t *testing.T) {
		t.Parallel()

		// No "s1" in this set
		errTmpl := template.Must(template.New("").Parse(`{{define "only"}}only{{end}}`))

		cfg := config.DashboardConfig{
			Tiles: []config.Tile{{Title: "Template Fail", Template: "s1"}},
		}

		html, rerr := RenderCell(t.Context(), 0, cfg, errTmpl, map[string]any{})
		require.Error(t, rerr)
		assert.EqualError(t, rerr, `template: Template rendering failed (html/template: "s1" is undefined)`)
		assert.Empty(t, html)
	})

	t.Run("handles invalid index", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}ok{{end}}`))
		cfg := config.DashboardConfig{} // no tiles

		html, rerr := RenderCell(t.Context(), 42, cfg, tmpl, map[string]any{})
		require.Error(t, rerr)
		assert.EqualError(t, rerr, "render: Failed to get tile (index out of range)")
		assert.Empty(t, html)
	})
}

func TestHasKey(t *testing.T) {
	t.Parallel()

	t.Run("present key with non-nil value", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"a": 1}
		assert.True(t, hasKey(m, "a"))
	})

	t.Run("present key with nil value", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"a": nil}
		assert.True(t, hasKey(m, "a"))
	})

	t.Run("absent key", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"a": 1}
		assert.False(t, hasKey(m, "b"))
	})
}

func TestNormalizeData(t *testing.T) {
	t.Parallel()

	t.Run("bytes -> json map", func(t *testing.T) {
		t.Parallel()
		in := []byte(`{"k":"v"}`)
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)
		require.Nil(t, acc)
		assert.Equal(t, map[string]any{"k": "v"}, primary)

		// Current implementation recurses and sets raw to the *decoded* value, not the original []byte.
		assert.NotNil(t, raw)
		_, ok := raw.(map[string]any)
		assert.True(t, ok)
	})

	t.Run("bytes -> invalid json", func(t *testing.T) {
		t.Parallel()
		in := []byte(`{oops}`)
		primary, acc, raw, err := normalizeData(in)
		require.Error(t, err)
		assert.Nil(t, primary)
		assert.Nil(t, acc)
		// raw should be the original bytes when unmarshal fails
		assert.True(t, reflect.DeepEqual(raw, in))
	})

	t.Run("plain map (single page object)", func(t *testing.T) {
		t.Parallel()
		in := map[string]any{"x": 1}
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)
		assert.Equal(t, in, primary)
		assert.Nil(t, acc)
		assert.Equal(t, in, raw)
	})

	t.Run("accumulator with merged preferred", func(t *testing.T) {
		t.Parallel()
		merged := map[string]any{"issues": []any{1, 2}}
		in := map[string]any{
			"merged": merged,
			"pages":  []map[string]any{{"p": 1}},
		}
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)
		assert.Equal(t, merged, primary)
		assert.Equal(t, in, acc)
		assert.Equal(t, in, raw)
	})

	t.Run("accumulator with pages []map, no merged", func(t *testing.T) {
		t.Parallel()
		p0 := map[string]any{"p": 0}
		in := map[string]any{
			"pages": []map[string]any{p0, {"p": 1}},
		}
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)
		assert.Equal(t, p0, primary)
		assert.Equal(t, in, acc)
		assert.Equal(t, in, raw)
	})

	t.Run("accumulator with pages []any (first map), no merged", func(t *testing.T) {
		t.Parallel()
		p0 := map[string]any{"p": 0}
		in := map[string]any{
			"pages": []any{p0, "ignored"},
		}
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)
		assert.Equal(t, p0, primary)
		assert.Equal(t, in, acc)
		assert.Equal(t, in, raw)
	})

	t.Run("accumulator with empty pages and no merged -> fallback to whole object", func(t *testing.T) {
		t.Parallel()
		in := map[string]any{
			"pages":  []map[string]any{},
			"other":  123,
			"merged": map[string]any{}, // empty
		}
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)
		assert.Equal(t, in, primary)
		assert.Equal(t, in, acc)
		assert.Equal(t, in, raw)
	})

	t.Run("named map type (reflection path)", func(t *testing.T) {
		t.Parallel()
		type Acc map[string]any
		in := Acc{
			"merged": map[string]any{"ok": true},
			"pages":  []map[string]any{{"p": 1}},
		}
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)

		assert.Equal(t, map[string]any{"ok": true}, primary)
		assert.Equal(t, map[string]any{
			"merged": map[string]any{"ok": true},
			"pages":  []map[string]any{{"p": 1}},
		}, acc)

		// raw is the normalized plain map (due to recursion), not the original Acc
		assert.Equal(t, map[string]any{
			"merged": map[string]any{"ok": true},
			"pages":  []map[string]any{{"p": 1}},
		}, raw)
	})

	t.Run("slice passthrough", func(t *testing.T) {
		t.Parallel()
		in := []int{1, 2, 3}
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)
		assert.True(t, reflect.DeepEqual(in, primary))
		assert.Nil(t, acc)
		assert.True(t, reflect.DeepEqual(in, raw))
	})

	t.Run("struct passthrough", func(t *testing.T) {
		t.Parallel()
		type S struct{ A int }
		in := S{A: 7}
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)
		assert.Equal(t, in, primary)
		assert.Nil(t, acc)
		assert.Equal(t, in, raw)
	})

	t.Run("primitive round-trip", func(t *testing.T) {
		t.Parallel()
		in := "hello"
		primary, acc, raw, err := normalizeData(in)
		require.NoError(t, err)
		assert.Equal(t, in, primary)
		assert.Nil(t, acc)
		assert.Equal(t, in, raw)
	})

	t.Run("json round-trip fallback for non-jsonable types returns error", func(t *testing.T) {
		t.Parallel()
		ch := make(chan int) // json.Marshal will error on channels
		primary, acc, raw, err := normalizeData(ch)
		require.Error(t, err)
		assert.Nil(t, primary)
		assert.Nil(t, acc)
		assert.Equal(t, any(ch), raw)
	})

	t.Run("bytes -> nested accumulator path (json types)", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{
			"merged": map[string]any{"k": "v"},
			"pages":  []map[string]any{{"p": 1}},
		}
		b, _ := json.Marshal(payload)

		primary, acc, raw, err := normalizeData(b)
		require.NoError(t, err)

		// primary should be the merged object
		assert.Equal(t, map[string]any{"k": "v"}, primary)

		// acc is decoded via encoding/json, so numbers are float64 and slices are []any
		gotPages, ok := acc["pages"].([]any)
		require.True(t, ok)
		require.Len(t, gotPages, 1)
		first, ok := gotPages[0].(map[string]any)
		require.True(t, ok)
		// number is float64
		_, isFloat := first["p"].(float64)
		assert.True(t, isFloat)

		// raw is the inner decoded value (current impl), not original bytes
		assert.NotNil(t, raw)
	})
}
