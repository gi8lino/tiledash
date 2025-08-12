package cache_test

import (
	"sync"
	"testing"
	"time"

	"github.com/gi8lino/tiledash/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemCache(t *testing.T) {
	t.Parallel()

	t.Run("Get on empty cache returns false", func(t *testing.T) {
		t.Parallel()
		m := cache.NewMemCache()
		_, ok := m.Get("missing")
		assert.False(t, ok)
	})

	t.Run("Set then Get returns a copy; modifying the returned map doesn't affect cache", func(t *testing.T) {
		t.Parallel()
		m := cache.NewMemCache()
		in := map[string]any{"x": 1}
		m.Set("k", in, time.Minute)

		// First retrieval
		got1, ok := m.Get("k")
		require.True(t, ok)
		assert.Equal(t, 1, got1["x"])

		// Mutate the map returned by Get — should NOT affect cached value
		got1["x"] = 999
		got2, ok := m.Get("k")
		require.True(t, ok)
		assert.Equal(t, 1, got2["x"], "cache should not reflect mutations on a previously returned copy")
	})

	t.Run("Set stores a shallow copy; mutating original top-level after Set does not affect cache", func(t *testing.T) {
		t.Parallel()
		m := cache.NewMemCache()
		in := map[string]any{
			"x": 1,
			// Note: nested maps are not deep-copied (documented limitation)
			"nested": map[string]any{"y": 2},
		}
		m.Set("k", in, time.Minute)

		// Mutate top-level fields on the original after Set
		in["x"] = 12345
		in["new"] = "added"

		got, ok := m.Get("k")
		require.True(t, ok)
		assert.Equal(t, 1, got["x"], "top-level field should be isolated via shallow copy on Set")
		assert.NotContains(t, got, "new", "new top-level keys added to original should not appear in cached value")

		// (Optional) Document shallow-copy behavior for nested structures:
		// If we mutate the original nested map, the cache may observe it because Set is shallow.
		origNested := in["nested"].(map[string]any)
		origNested["y"] = 777 // mutate nested map in the original

		got2, ok := m.Get("k")
		require.True(t, ok)
		// Depending on desired semantics, this shows current implementation limitation:
		assert.Equal(t, 777, got2["nested"].(map[string]any)["y"], "nested maps share references due to shallow copies")
	})

	t.Run("TTL expiry evicts entries lazily", func(t *testing.T) {
		t.Parallel()
		m := cache.NewMemCache()
		m.Set("soon", map[string]any{"v": 1}, 30*time.Millisecond)

		// Immediately available
		_, ok := m.Get("soon")
		require.True(t, ok)

		// After expiry
		time.Sleep(40 * time.Millisecond)
		_, ok = m.Get("soon")
		assert.False(t, ok, "expired entry should be evicted on access")
	})

	t.Run("negative TTL results in immediate expiry", func(t *testing.T) {
		t.Parallel()
		m := cache.NewMemCache()
		m.Set("neg", map[string]any{"v": 1}, -1*time.Nanosecond)
		_, ok := m.Get("neg")
		assert.False(t, ok)
	})

	t.Run("concurrent Set/Get is safe", func(t *testing.T) {
		t.Parallel()
		m := cache.NewMemCache()

		var wg sync.WaitGroup
		keys := []string{"a", "b", "c", "d", "e"}

		for _, k := range keys {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range 200 {
					m.Set(k, map[string]any{"i": i}, time.Second)
					_, _ = m.Get(k) // best-effort read; we just care that it doesn't race/panic
				}
			}()
		}
		wg.Wait()

		// Final read: should exist and be a map with an "i" field (last write wins, value is unspecified)
		for _, k := range keys {
			got, ok := m.Get(k)
			require.True(t, ok, "expected key %q to exist", k)
			_, present := got["i"]
			assert.True(t, present, "expected 'i' field to exist for key %q", k)
		}
	})
}
