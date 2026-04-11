package broker

import (
	"context"
	"log/slog"
	"sync"

	"nhooyr.io/websocket"
)

// WSClient represents a single WebSocket connection belonging to an admin
// user in a specific session.
type WSClient struct {
	conn      *websocket.Conn
	sessionID string
}

// WSHub manages session-scoped WebSocket connections for admin users. Each
// session maintains its own set of clients so that broadcast messages are
// delivered only to admins of the target session, never to other sessions.
// The hub is safe for concurrent use.
type WSHub struct {
	mu      sync.RWMutex
	clients map[string]map[*WSClient]struct{} // sessionID -> set of clients
}

// NewWSHub creates a WSHub with an empty client map, ready to accept
// registrations.
func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[string]map[*WSClient]struct{}),
	}
}

// Register adds a new WebSocket connection to the hub for the given session
// and returns the resulting WSClient handle (used later for Unregister).
func (h *WSHub) Register(sessionID string, conn *websocket.Conn) *WSClient {
	h.mu.Lock()
	defer h.mu.Unlock()

	client := &WSClient{conn: conn, sessionID: sessionID}
	if h.clients[sessionID] == nil {
		h.clients[sessionID] = make(map[*WSClient]struct{})
	}
	h.clients[sessionID][client] = struct{}{}

	count := len(h.clients[sessionID])
	slog.Info("WebSocket client connected", "sessionId", sessionID, "adminCount", count)

	return client
}

// Unregister removes a client from the hub. If the session has no remaining
// clients after removal, its entry is cleaned up entirely.
func (h *WSHub) Unregister(client *WSClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.sessionID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.clients, client.sessionID)
		}
	}

	count := len(h.clients[client.sessionID])
	slog.Info("WebSocket client disconnected", "sessionId", client.sessionID, "adminCount", count)
}

// Broadcast sends a message to every admin client connected to the given
// session. The client list is snapshot under a read lock and then released
// before writing, so a slow or blocked write does not hold the hub lock and
// starve other operations.
func (h *WSHub) Broadcast(sessionID string, message []byte) {
	h.mu.RLock()
	clients, ok := h.clients[sessionID]
	if !ok {
		h.mu.RUnlock()
		return
	}
	// Snapshot client list so we can release the lock before writing.
	targets := make([]*WSClient, 0, len(clients))
	for client := range clients {
		targets = append(targets, client)
	}
	h.mu.RUnlock()

	for _, client := range targets {
		err := client.conn.Write(context.Background(), websocket.MessageText, message)
		if err != nil {
			slog.Warn("failed to send WebSocket message",
				"sessionId", sessionID,
				"err", err,
			)
		}
	}

	slog.Debug("WebSocket message broadcast",
		"sessionId", sessionID,
		"recipientCount", len(targets),
	)
}
