package access

import (
	"context"
	"testing"
	"time"
)

func TestNewAccessCache(t *testing.T) {
	c := NewAccessCache(5 * time.Minute)
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
}

func TestAccessCache_GetSet(t *testing.T) {
	c := NewAccessCache(5 * time.Minute)

	// Get missing key
	val, ok := c.Get("missing")
	if ok {
		t.Error("expected not found for missing key")
	}
	if val {
		t.Error("expected false for missing key")
	}

	// Set and get
	c.Set("key1", true)
	val, ok = c.Get("key1")
	if !ok {
		t.Error("expected found after set")
	}
	if !val {
		t.Error("expected true for key1")
	}

	c.Set("key2", false)
	val, ok = c.Get("key2")
	if !ok {
		t.Error("expected found for key2")
	}
	if val {
		t.Error("expected false for key2")
	}
}

func TestAccessCache_Expired(t *testing.T) {
	c := NewAccessCache(1 * time.Millisecond)
	c.Set("key1", true)
	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected expired entry to not be found")
	}
}

func TestAccessCache_Invalidate(t *testing.T) {
	c := NewAccessCache(5 * time.Minute)
	c.Set("key1", true)
	c.Invalidate("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected invalidated key to not be found")
	}
}

func TestAccessCache_InvalidateAll(t *testing.T) {
	c := NewAccessCache(5 * time.Minute)
	c.Set("key1", true)
	c.Set("key2", false)
	c.InvalidateAll()

	_, ok1 := c.Get("key1")
	_, ok2 := c.Get("key2")
	if ok1 || ok2 {
		t.Error("expected all keys invalidated")
	}
}

func TestNewTierGater(t *testing.T) {
	g := NewTierGater()
	if g == nil {
		t.Fatal("expected non-nil gater")
	}
}

func TestTierGater_VerifyAccess_Granted(t *testing.T) {
	g := NewTierGater()
	ctx := context.Background()

	ok, url, err := g.VerifyAccess(ctx, "patron1", "content1", "premium", []string{"basic", "premium"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected access granted")
	}
	if url != "" {
		t.Errorf("expected empty upgrade URL, got %q", url)
	}
}

func TestTierGater_VerifyAccess_Denied(t *testing.T) {
	g := NewTierGater()
	ctx := context.Background()

	ok, url, err := g.VerifyAccess(ctx, "patron1", "content1", "premium", []string{"basic"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected access denied")
	}
	if url == "" {
		t.Error("expected non-empty upgrade URL")
	}
}

func TestTierGater_VerifyAccess_CachedGranted(t *testing.T) {
	g := NewTierGater()
	ctx := context.Background()

	// First call populates cache
	g.VerifyAccess(ctx, "patron1", "content1", "premium", []string{"premium"})

	// Second call uses cache (granted)
	ok, url, err := g.VerifyAccess(ctx, "patron1", "content1", "premium", []string{"premium"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected cached access granted")
	}
	if url != "" {
		t.Errorf("expected empty URL for cached grant, got %q", url)
	}
}

func TestTierGater_VerifyAccess_CachedDenied(t *testing.T) {
	g := NewTierGater()
	ctx := context.Background()

	// First call populates cache (denied)
	g.VerifyAccess(ctx, "patron1", "content1", "premium", []string{"basic"})

	// Second call uses cache (denied)
	ok, url, err := g.VerifyAccess(ctx, "patron1", "content1", "premium", []string{"basic"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected cached access denied")
	}
	if url == "" {
		t.Error("expected non-empty upgrade URL for cached denial")
	}
}
