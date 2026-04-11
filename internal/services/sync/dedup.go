package sync

import (
	"sync"
	"time"
)

type EventDeduplicator struct {
	mu     sync.RWMutex
	seen   map[string]time.Time
	window time.Duration
	stop   chan struct{}
	done   chan struct{}
}

func NewEventDeduplicator(window time.Duration) *EventDeduplicator {
	ed := &EventDeduplicator{
		seen:   make(map[string]time.Time),
		window: window,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go ed.cleanup()
	return ed
}

func (ed *EventDeduplicator) TrackEvent(eventID string) {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	ed.seen[eventID] = time.Now()
}

func (ed *EventDeduplicator) IsDuplicate(eventID string) bool {
	ed.mu.RLock()
	defer ed.mu.RUnlock()
	t, exists := ed.seen[eventID]
	if !exists {
		return false
	}
	return time.Since(t) < ed.window
}

func (ed *EventDeduplicator) cleanup() {
	defer close(ed.done)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ed.stop:
			return
		case <-ticker.C:
			ed.mu.Lock()
			now := time.Now()
			for id, t := range ed.seen {
				if now.Sub(t) > ed.window {
					delete(ed.seen, id)
				}
			}
			ed.mu.Unlock()
		}
	}
}

func (ed *EventDeduplicator) Close() error {
	close(ed.stop)
	select {
	case <-ed.done:
		return nil
	case <-time.After(1 * time.Second):
		return ErrShutdownTimeout
	}
}
