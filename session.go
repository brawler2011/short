package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

// WSMessage is a typed JSON message sent over WebSocket.
type WSMessage struct {
	Type        string `json:"type"`
	Token       string `json:"token,omitempty"`
	SessionURL  string `json:"session_url,omitempty"`
	ShortURL    string `json:"short_url,omitempty"`
	OriginalURL string `json:"original_url,omitempty"`
	Message     string `json:"message,omitempty"`
}

// Hub manages active WebSocket sessions.
type Hub struct {
	mu   sync.RWMutex
	cons map[string]*websocket.Conn // token → conn
}

func NewHub() *Hub {
	return &Hub{cons: make(map[string]*websocket.Conn)}
}

// Register associates a WebSocket connection with a session token.
func (h *Hub) Register(token string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cons[token] = conn
}

// Unregister removes a connection by token.
func (h *Hub) Unregister(token string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.cons, token)
}

// Push sends a message to the computer holding the given session token.
func (h *Hub) Push(token string, msg WSMessage) error {
	h.mu.RLock()
	conn, ok := h.cons[token]
	h.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session not connected: %s", token)
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

// HasSession returns true if the token has an active WS connection.
func (h *Hub) HasSession(token string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.cons[token]
	return ok
}
