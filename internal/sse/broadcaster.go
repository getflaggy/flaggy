package sse

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

// Event represents an SSE event sent to connected clients.
type Event struct {
	ID   string `json:"id"`
	Type string `json:"type"` // flag_created, flag_updated, flag_deleted, flag_toggled, rule_created, rule_updated, rule_deleted
	Data any    `json:"data"`
}

// Broadcaster fans out events to all connected SSE clients.
type Broadcaster struct {
	mu      sync.RWMutex
	clients map[uint64]chan Event
	nextID  atomic.Uint64
}

// NewBroadcaster creates a new SSE broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[uint64]chan Event),
	}
}

// Subscribe registers a new client and returns a channel to receive events
// and a function to unsubscribe.
func (b *Broadcaster) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 64)
	id := b.nextID.Add(1)

	b.mu.Lock()
	b.clients[id] = ch
	b.mu.Unlock()

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.clients[id]; ok {
			delete(b.clients, id)
			close(ch)
		}
	}

	return ch, unsub
}

// Publish sends an event to all connected clients.
// Non-blocking: if a client's buffer is full, the event is dropped for that client.
func (b *Broadcaster) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.clients {
		select {
		case ch <- event:
		default:
			// Client too slow, drop event
		}
	}
}

// ClientCount returns the number of connected clients.
func (b *Broadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Close closes all client channels.
func (b *Broadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, ch := range b.clients {
		close(ch)
		delete(b.clients, id)
	}
}

// MarshalData is a helper to create a JSON data payload.
func MarshalData(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
