package audit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEntryValidateRequiresActor(t *testing.T) {
	if err := (Entry{Action: "sync"}).Validate(); err == nil {
		t.Fatal("expected error for missing Actor")
	}
}

func TestEntryValidateRequiresAction(t *testing.T) {
	if err := (Entry{Actor: "cli"}).Validate(); err == nil {
		t.Fatal("expected error for missing Action")
	}
}

func TestEntryValidatePassesWhenRequiredSet(t *testing.T) {
	if err := (Entry{Actor: "cli", Action: "sync"}).Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRingStoreKeepsLastN(t *testing.T) {
	r := NewRingStore(3)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := r.Write(ctx, Entry{
			Actor:     "cli",
			Action:    "sync",
			Target:    "repo-" + string(rune('0'+i)),
			CreatedAt: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := r.List(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("len = %d, want 3", len(entries))
	}
	if entries[0].Target != "repo-2" || entries[2].Target != "repo-4" {
		t.Fatalf("ring order wrong: %+v", entries)
	}
}

func TestRingStoreListLimit(t *testing.T) {
	r := NewRingStore(10)
	for i := 0; i < 5; i++ {
		_ = r.Write(context.Background(), Entry{Actor: "a", Action: "b"})
	}
	entries, _ := r.List(context.Background(), 2)
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
}

func TestRingStoreRejectsInvalid(t *testing.T) {
	r := NewRingStore(3)
	if err := r.Write(context.Background(), Entry{}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRingStoreConcurrent(t *testing.T) {
	r := NewRingStore(100)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = r.Write(context.Background(), Entry{Actor: "a", Action: "b"})
			}
		}()
	}
	wg.Wait()
	entries, _ := r.List(context.Background(), 0)
	if len(entries) != 100 {
		t.Fatalf("len = %d, want 100 (bounded)", len(entries))
	}
}

func TestNewRingStoreClampsSize(t *testing.T) {
	r := NewRingStore(0)
	if r.size != 1 {
		t.Fatalf("size = %d, want 1", r.size)
	}
}

func TestRingStoreCloseNoop(t *testing.T) {
	r := NewRingStore(3)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}
