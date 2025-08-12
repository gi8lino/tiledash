package cache

import (
	"maps"
	"sync"
	"time"
)

// MemCache is a minimal TTL map[string] -> JSON object cache.
type MemCache struct {
	mu   sync.RWMutex
	data map[string]memItem
}

// memItem stores a value and its expiry time.
type memItem struct {
	val   map[string]any
	expAt time.Time
}

// NewMemCache constructs an in-memory TTL cache.
func NewMemCache() *MemCache { return &MemCache{data: make(map[string]memItem)} }

// Get retrieves a cached value if not expired.
func (m *MemCache) Get(key string) (map[string]any, bool) {
	m.mu.RLock()
	item, ok := m.data[key]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(item.expAt) {
		m.mu.Lock()
		delete(m.data, key)
		m.mu.Unlock()
		return nil, false
	}
	// return a shallow copy to avoid callers mutating cached map
	out := make(map[string]any, len(item.val))
	maps.Copy(out, item.val)
	return out, true
}

// Set stores a value with TTL.
func (m *MemCache) Set(key string, v map[string]any, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make(map[string]any, len(v))
	maps.Copy(cp, v)
	m.data[key] = memItem{val: cp, expAt: time.Now().Add(ttl)}
}
