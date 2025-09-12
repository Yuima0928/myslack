package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu       sync.RWMutex
	channels map[string]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{channels: map[string]map[*websocket.Conn]struct{}{}}
}

func (h *Hub) Join(channel string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.channels[channel] == nil {
		h.channels[channel] = map[*websocket.Conn]struct{}{}
	}
	h.channels[channel][conn] = struct{}{}
}

func (h *Hub) Leave(channel string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.channels[channel], conn)
}

func (h *Hub) Broadcast(channel string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.channels[channel] {
		_ = c.WriteMessage(websocket.TextMessage, payload)
	}
}
