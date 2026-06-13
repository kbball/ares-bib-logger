package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Publisher is the interface adapters use to push events to connected SSE clients.
type Publisher interface {
	Publish(eventType string, payload any)
}

type envelope struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Broker manages SSE client connections and broadcasts events to all of them.
type Broker struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func NewBroker() *Broker {
	return &Broker{clients: make(map[chan string]struct{})}
}

// Publish sends an event to all connected SSE clients. Non-blocking — slow clients are skipped.
func (b *Broker) Publish(eventType string, payload any) {
	data, err := json.Marshal(envelope{Type: eventType, Payload: payload})
	if err != nil {
		return
	}
	msg := fmt.Sprintf("data: %s\n\n", data)

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (b *Broker) subscribe() (chan string, func()) {
	ch := make(chan string, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.clients, ch)
		b.mu.Unlock()
		close(ch)
	}
}

// ServeHTTP handles GET /api/stream — registers the client and streams events until disconnect.
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsubscribe := b.subscribe()
	defer unsubscribe()

	_, _ = fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprint(w, msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
