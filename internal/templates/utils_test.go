package templates

import (
	"html/template"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateFuncMap(t *testing.T) {
	t.Parallel()
	fm := TemplateFuncMap()

	// Sprig base exists (spot-check)
	require.NotNil(t, fm)

	// Our customs are all present
	for _, k := range []string{
		"formatJiraDate",
		"setany",
		"dig",
		"sortBy",
		"appendSlice",
		"uniq",
		"defaultStr",
		"typeOf",
		"sumBy",
	} {
		_, ok := fm[k]
		assert.Truef(t, ok, "func %q should be present", k)
	}
}

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

	t.Run("sorts by int64 field ascending (hits int64/compareFloat64 path)", func(t *testing.T) {
		t.Parallel()
		input := []any{
			map[string]any{"n": int64(5)},
			map[string]any{"n": int64(2)},
			map[string]any{"n": int64(3)},
		}
		sorted := sortBy("n", false, input) // asc
		assert.Equal(t, int64(2), sorted[0].(map[string]any)["n"])
		assert.Equal(t, int64(3), sorted[1].(map[string]any)["n"])
		assert.Equal(t, int64(5), sorted[2].(map[string]any)["n"])
	})

	t.Run("sorts by string field ascending (hits vi < vjs path)", func(t *testing.T) {
		t.Parallel()
		input := []any{
			map[string]any{"name": "Charlie"},
			map[string]any{"name": "Alice"},
			map[string]any{"name": "Bob"},
		}
		sorted := sortBy("name", false, input) // asc
		assert.Equal(t, "Alice", sorted[0].(map[string]any)["name"])
		assert.Equal(t, "Bob", sorted[1].(map[string]any)["name"])
		assert.Equal(t, "Charlie", sorted[2].(map[string]any)["name"])
	})

	t.Run("sorts by time field ascending (hits Before path)", func(t *testing.T) {
		t.Parallel()
		base := time.Unix(1_700_000_000, 0).UTC()
		input := []any{
			map[string]any{"ts": base.Add(2 * time.Hour)},
			map[string]any{"ts": base.Add(1 * time.Hour)},
			map[string]any{"ts": base},
		}
		sorted := sortBy("ts", false, input) // asc -> Before
		assert.Equal(t, base, sorted[0].(map[string]any)["ts"])
		assert.Equal(t, base.Add(1*time.Hour), sorted[1].(map[string]any)["ts"])
		assert.Equal(t, base.Add(2*time.Hour), sorted[2].(map[string]any)["ts"])
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

	t.Run("stable when field missing/unknown type", func(t *testing.T) {
		t.Parallel()
		input := []any{
			map[string]any{"x": struct{}{}},
			map[string]any{"x": struct{}{}},
		}
		sorted := sortBy("x", false, input)
		assert.Equal(t, input, sorted) // stable sort + comparator false keeps order
	})

	t.Run("panics when not []any", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			_ = sortBy("x", false, "not-a-slice")
		})
	})
}

func TestCompareFloat64(t *testing.T) {
	t.Parallel()

	t.Run("compares against int ascending", func(t *testing.T) {
		t.Parallel()
		assert.True(t, compareFloat64(1, 2, false))
		assert.False(t, compareFloat64(3, 2, false))
	})

	t.Run("compares against int64 descending", func(t *testing.T) {
		t.Parallel()
		assert.True(t, compareFloat64(5, int64(4), true))
		assert.False(t, compareFloat64(3, int64(4), true))
	})

	t.Run("compares against float64 ascending", func(t *testing.T) {
		t.Parallel()
		assert.True(t, compareFloat64(1.5, 2.0, false))
	})

	t.Run("returns false for non-numeric vj", func(t *testing.T) {
		t.Parallel()
		assert.False(t, compareFloat64(1.0, "x", false))
	})
}

func TestUniq(t *testing.T) {
	t.Parallel()

	t.Run("removes duplicates, preserves order of first appearance", func(t *testing.T) {
		t.Parallel()
		in := []string{"a", "b", "a", "c", "b", "d", "d"}
		out := uniq(in)
		assert.Equal(t, []string{"a", "b", "c", "d"}, out)
	})

	t.Run("handles empty slice", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{}, uniq(nil))
	})
}

func TestDefaultStr(t *testing.T) {
	t.Parallel()

	t.Run("returns original when non-empty", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "x", defaultStr("x", "fallback"))
	})

	t.Run("returns fallback on empty", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "fb", defaultStr("", "fb"))
	})

	t.Run("returns fallback on whitespace", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "fb", defaultStr("   \t", "fb"))
	})
}

func TestTypeOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "int", typeOf(1))
	assert.Equal(t, "map[string]interface {}", typeOf(map[string]any{"x": 1}))
	assert.Equal(t, "template.HTML", typeOf(template.HTML("<b>x</b>")))
}

func TestSumBy(t *testing.T) {
	t.Parallel()

	t.Run("sums int, int64, float64", func(t *testing.T) {
		t.Parallel()
		items := []map[string]any{
			{"n": 1},
			{"n": int64(2)},
			{"n": 3.5},
			{"n": "skip"},
			{}, // missing
		}
		total := sumBy("n", items)
		assert.InDelta(t, 6.5, total, 1e-9)
	})

	t.Run("zero when field missing everywhere", func(t *testing.T) {
		t.Parallel()
		items := []map[string]any{{}, {}}
		assert.Equal(t, 0.0, sumBy("n", items))
	})
}
