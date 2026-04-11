package handlers

import (
	"log/slog"
	"net/http"

	"github.com/cuetv/backend/internal/broker"
	"github.com/cuetv/backend/internal/services"
	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
)

// WebSocketHandler handles WebSocket connections for real-time admin queue synchronization.
type WebSocketHandler struct {
	wsHub          *broker.WSHub
	authService    *services.AuthService
	allowedOrigins []string
}

// NewWebSocketHandler creates a new WebSocketHandler with the given WebSocket hub, auth service, and allowed origins.
func NewWebSocketHandler(wsHub *broker.WSHub, authService *services.AuthService, allowedOrigins []string) *WebSocketHandler {
	return &WebSocketHandler{
		wsHub:          wsHub,
		authService:    authService,
		allowedOrigins: allowedOrigins,
	}
}

// HandleQueueEvents upgrades the connection to a WebSocket after validating the JWT,
// then keeps the connection alive to receive real-time queue update broadcasts.
func (h *WebSocketHandler) HandleQueueEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session ID", http.StatusBadRequest)
		return
	}

	// Validate JWT before upgrading
	token := r.URL.Query().Get("token")
	if token == "" {
		slog.Warn("WebSocket connection attempted without token", "sessionId", sessionID)
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.authService.ValidateToken(token)
	if err != nil || claims.SessionID != sessionID {
		slog.Warn("invalid token on WebSocket connection", "sessionId", sessionID, "err", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	if claims.Role != "admin" {
		slog.Warn("non-admin attempted WebSocket connection", "sessionId", sessionID, "role", claims.Role)
		http.Error(w, "forbidden: admin role required", http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.allowedOrigins,
	})
	if err != nil {
		slog.Error("failed to accept WebSocket", "err", err)
		return
	}

	client := h.wsHub.Register(sessionID, conn)
	defer h.wsHub.Unregister(client)

	// Keep connection alive — read loop to detect disconnects
	for {
		_, _, err := conn.Read(r.Context())
		if err != nil {
			slog.Debug("WebSocket client disconnected", "sessionId", sessionID)
			return
		}
	}
}
