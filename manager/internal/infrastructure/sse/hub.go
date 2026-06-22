package sse

import (
	"encoding/json"
	"sync"
)

// Client represents a connected SSE client.
type Client struct {
	ID     string
	Events chan []byte
}

// Hub manages SSE client connections and broadcasts events to all subscribers.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]struct{}
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	done       chan struct{}
}

// NewHub creates and starts a new SSE hub.
func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
	}
	go h.run()
	return h
}

// run is the main event loop that handles client registration, unregistration,
// and broadcasting messages to all connected clients.
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = struct{}{}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Events)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Events <- msg:
				default:
					// Client buffer full, skip this message to avoid blocking.
				}
			}
			h.mu.RUnlock()

		case <-h.done:
			h.mu.Lock()
			for client := range h.clients {
				close(client.Events)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		}
	}
}

// Register adds a new client to the hub and returns it.
func (h *Hub) Register(clientID string) *Client {
	client := &Client{
		ID:     clientID,
		Events: make(chan []byte, 64),
	}
	h.register <- client
	return client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a JSON-encoded message to all connected clients.
// Non-blocking; drops message if broadcast channel is full to prevent
// blocking worker goroutines.
func (h *Hub) Broadcast(eventType string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	msg := formatSSEMessage(eventType, payload)
	select {
	case h.broadcast <- msg:
	default:
	}
}

// Stop shuts down the hub gracefully.
func (h *Hub) Stop() {
	close(h.done)
}

// ClientCount returns the number of currently connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// formatSSEMessage formats data as an SSE message with event type.
func formatSSEMessage(eventType string, data []byte) []byte {
	var msg []byte
	if eventType != "" {
		msg = append(msg, []byte("event: "+eventType+"\n")...)
	}
	msg = append(msg, []byte("data: ")...)
	msg = append(msg, data...)
	msg = append(msg, []byte("\n\n")...)
	return msg
}
