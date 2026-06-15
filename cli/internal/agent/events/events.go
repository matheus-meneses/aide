package events

import (
	"aide/cli/internal/platform/clog"
	"sync"
	"sync/atomic"
	"time"
)

var elog = clog.New("events")

type Event struct {
	ID        uint64 `json:"id"`
	Type      string `json:"type"`
	Data      string `json:"data"`
	Timestamp string `json:"timestamp"`
	Priority  string `json:"priority,omitempty"`
}

type EventRing struct {
	mu     sync.RWMutex
	events []Event
	cap    int
	head   int
	count  int
}

func NewEventRing(capacity int) *EventRing {
	return &EventRing{
		events: make([]Event, capacity),
		cap:    capacity,
	}
}

func (r *EventRing) Push(e Event) {
	r.mu.Lock()
	r.events[r.head] = e
	r.head = (r.head + 1) % r.cap
	if r.count < r.cap {
		r.count++
	}
	r.mu.Unlock()
}

func (r *EventRing) Since(afterID uint64) []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Event
	start := r.head - r.count
	if start < 0 {
		start += r.cap
	}
	for i := 0; i < r.count; i++ {
		idx := (start + i) % r.cap
		if r.events[idx].ID > afterID {
			result = append(result, r.events[idx])
		}
	}
	return result
}

func (r *EventRing) Recent(n int) []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := n
	if count > r.count {
		count = r.count
	}
	result := make([]Event, count)
	start := r.head - count
	if start < 0 {
		start += r.cap
	}
	for i := 0; i < count; i++ {
		idx := (start + i) % r.cap
		result[i] = r.events[idx]
	}
	return result
}

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
	ring        *EventRing
	nextID      atomic.Uint64
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[chan Event]struct{}),
		ring:        NewEventRing(500),
	}
}

func (b *EventBus) Subscribe() (ch <-chan Event, unsubscribe func()) {
	c := make(chan Event, 128)
	b.mu.Lock()
	b.subscribers[c] = struct{}{}
	b.mu.Unlock()

	return c, func() {
		b.mu.Lock()
		delete(b.subscribers, c)
		b.mu.Unlock()
		close(c)
	}
}

func (b *EventBus) Publish(e Event) {
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	e.ID = b.nextID.Add(1)
	b.ring.Push(e)

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- e:
		default:
			elog.Warn("sse dropped event id=%d type=%s for slow subscriber", e.ID, e.Type)
		}
	}
}

func (b *EventBus) Ring() *EventRing {
	return b.ring
}
