package cache

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
)

func TestTTLLRU_GetAfterAdd(t *testing.T) {
	c := NewTTLLRU[string, int](10, 5*time.Minute)
	c.Add("a", 1)
	v, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)
}

func TestTTLLRU_Miss(t *testing.T) {
	c := NewTTLLRU[string, int](10, 5*time.Minute)
	_, ok := c.Get("missing")
	assert.False(t, ok)
}

func TestTTLLRU_Expiry(t *testing.T) {
	fake := clockwork.NewFakeClock()
	c := NewTTLLRU[string, int](10, 5*time.Minute)
	c.now = fake.Now

	c.Add("k", 42)

	v, ok := c.Get("k")
	assert.True(t, ok)
	assert.Equal(t, 42, v)

	fake.Advance(6 * time.Minute)

	_, ok = c.Get("k")
	assert.False(t, ok, "entry must expire after TTL")
}

func TestTTLLRU_Eviction(t *testing.T) {
	c := NewTTLLRU[string, int](2, time.Hour)
	c.Add("a", 1)
	c.Add("b", 2)
	c.Add("c", 3) // evicts "a"

	_, ok := c.Get("a")
	assert.False(t, ok, "oldest entry must be evicted")

	v, ok := c.Get("c")
	assert.True(t, ok)
	assert.Equal(t, 3, v)
}

func TestTTLLRU_Overwrite(t *testing.T) {
	c := NewTTLLRU[string, int](10, time.Hour)
	c.Add("k", 1)
	c.Add("k", 2)
	v, ok := c.Get("k")
	assert.True(t, ok)
	assert.Equal(t, 2, v)
}
