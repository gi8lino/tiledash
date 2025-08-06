package templates

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppendSlice(t *testing.T) {
	t.Parallel()

	t.Run("appends to empty []any", func(t *testing.T) {
		t.Parallel()
		var input []any
		result := appendSlice(input, "value")
		assert.Equal(t, []any{"value"}, result)
	})

	t.Run("appends to prefilled []any", func(t *testing.T) {
		t.Parallel()
		input := []any{"a", "b"}
		result := appendSlice(input, "c")
		assert.Equal(t, []any{"a", "b", "c"}, result)
	})

	t.Run("wraps non-slice as single-element", func(t *testing.T) {
		t.Parallel()
		result := appendSlice("foo", "bar")
		assert.Equal(t, []any{"bar"}, result)
	})
}

func TestTemplateDig(t *testing.T) {
	t.Parallel()

	t.Run("returns value from map[string]any if key exists and is string", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"foo": "bar"}
		assert.Equal(t, "bar", templateDig(m, "foo"))
	})

	t.Run("returns empty string if key missing", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"foo": "bar"}
		assert.Equal(t, "", templateDig(m, "baz"))
	})

	t.Run("returns empty string if value is not string", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"foo": 42}
		assert.Equal(t, "", templateDig(m, "foo"))
	})

	t.Run("returns input if already string", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello", templateDig("hello", "ignored"))
	})

	t.Run("returns empty string for unknown type", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", templateDig(123, "ignored"))
	})
}

func TestSetAny(t *testing.T) {
	t.Parallel()

	t.Run("sets new key", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{}
		setany(m, "k", 123)
		assert.Equal(t, 123, m["k"])
	})

	t.Run("overwrites existing key", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"k": "old"}
		setany(m, "k", "new")
		assert.Equal(t, "new", m["k"])
	})

	t.Run("returns same map", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"x": "y"}
		res := setany(m, "z", "w")
		assert.Equal(t, m, res)
	})
}

func TestFormatJiraDate(t *testing.T) {
	t.Parallel()

	t.Run("parses valid Jira timestamp", func(t *testing.T) {
		t.Parallel()
		input := "2024-08-01T12:34:56.789+0200"
		layout := "2006-01-02 15:04"
		expected := "2024-08-01 12:34"
		actual := formatJiraDate(input, layout)
		assert.Equal(t, expected, actual)
	})

	t.Run("parses valid Zulu time and normalizes", func(t *testing.T) {
		t.Parallel()
		input := "2024-08-01T10:00:00.000Z"
		layout := time.RFC3339
		actual := formatJiraDate(input, layout)
		assert.Contains(t, actual, "2024-08-01")
	})

	t.Run("returns input on parse failure", func(t *testing.T) {
		t.Parallel()
		input := "invalid"
		layout := "irrelevant"
		assert.Equal(t, "invalid", formatJiraDate(input, layout))
	})
}

func TestSortBy(t *testing.T) {
	t.Parallel()

	t.Run("sorts by int field ascending", func(t *testing.T) {
		t.Parallel()
		input := []any{
			map[string]any{"val": 3},
			map[string]any{"val": 1},
			map[string]any{"val": 2},
		}
		sorted := sortBy("val", false, input)
		assert.Equal(t, 1, sorted[0].(map[string]any)["val"])
		assert.Equal(t, 2, sorted[1].(map[string]any)["val"])
		assert.Equal(t, 3, sorted[2].(map[string]any)["val"])
	})

	t.Run("sorts by string field descending", func(t *testing.T) {
		t.Parallel()
		input := []any{
			map[string]any{"name": "Alice"},
			map[string]any{"name": "Charlie"},
			map[string]any{"name": "Bob"},
		}
		sorted := sortBy("name", true, input)
		assert.Equal(t, "Charlie", sorted[0].(map[string]any)["name"])
		assert.Equal(t, "Bob", sorted[1].(map[string]any)["name"])
		assert.Equal(t, "Alice", sorted[2].(map[string]any)["name"])
	})

	t.Run("sorts by float64 field ascending", func(t *testing.T) {
		t.Parallel()
		input := []any{
			map[string]any{"score": 3.2},
			map[string]any{"score": 1.1},
			map[string]any{"score": 2.5},
		}
		sorted := sortBy("score", false, input)
		assert.InDelta(t, 1.1, sorted[0].(map[string]any)["score"], 0.01)
		assert.InDelta(t, 2.5, sorted[1].(map[string]any)["score"], 0.01)
		assert.InDelta(t, 3.2, sorted[2].(map[string]any)["score"], 0.01)
	})

	t.Run("sorts by time field descending", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		input := []any{
			map[string]any{"created": now.Add(-time.Hour)},
			map[string]any{"created": now},
			map[string]any{"created": now.Add(-2 * time.Hour)},
		}
		sorted := sortBy("created", true, input)
		assert.Equal(t, now, sorted[0].(map[string]any)["created"])
	})
}
