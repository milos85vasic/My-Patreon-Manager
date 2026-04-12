package cache

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// entry wraps a cached value with its insertion timestamp.
type entry[V any] struct {
	val     V
	created time.Time
}

// TTLLRU is a thread-safe LRU cache with per-entry time-to-live expiry.
type TTLLRU[K comparable, V any] struct {
	mu    sync.Mutex
	inner *lru.Cache[K, entry[V]]
	ttl   time.Duration
	now   func() time.Time // overridable for testing
}

// NewTTLLRU creates a new TTLLRU with the given maximum size and TTL.
func NewTTLLRU[K comparable, V any](size int, ttl time.Duration) *TTLLRU[K, V] {
	c, _ := lru.New[K, entry[V]](size)
	return &TTLLRU[K, V]{
		inner: c,
		ttl:   ttl,
		now:   time.Now,
	}
}

// Get returns the value for key and true if it exists and has not expired.
func (c *TTLLRU[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.inner.Get(key)
	if !ok {
		var zero V
		return zero, false
	}
	if c.now().Sub(e.created) > c.ttl {
		c.inner.Remove(key)
		var zero V
		return zero, false
	}
	return e.val, true
}

// Add inserts or updates a key/value pair. The TTL timer starts from now.
func (c *TTLLRU[K, V]) Add(key K, val V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.inner.Add(key, entry[V]{val: val, created: c.now()})
}
