package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/cuetv/backend/internal/middleware"
	"github.com/cuetv/backend/internal/models"
	"github.com/cuetv/backend/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// SessionHandler handles HTTP requests for session lifecycle operations.
type SessionHandler struct {
	sessionService *services.SessionService
	authService    *services.AuthService
}

// NewSessionHandler creates a new SessionHandler with the given services.
func NewSessionHandler(sessionService *services.SessionService, authService *services.AuthService) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		authService:    authService,
	}
}

// Create handles session creation after verifying the admin password.
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		slog.Warn("failed to decode create session request", "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !h.authService.VerifyAdminPassword(req.Password) {
		slog.Warn("invalid admin password on session create")
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	resp, err := h.sessionService.Create(r.Context())
	if err != nil {
		slog.Error("failed to create session", "err", err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Join handles joining an existing session using a friend key.
func (h *SessionHandler) Join(w http.ResponseWriter, r *http.Request) {
	var req models.JoinSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		slog.Warn("failed to decode join session request", "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.FriendKey == "" {
		slog.Warn("join session called without friend key")
		http.Error(w, "friend key is required", http.StatusBadRequest)
		return
	}

	resp, err := h.sessionService.Join(r.Context(), req.FriendKey)
	if err != nil {
		slog.Warn("session not found for friend key", "err", err)
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Rejoin handles re-joining a session using an existing JWT.
func (h *SessionHandler) Rejoin(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := h.sessionService.Rejoin(r.Context(), claims.SessionID)
	if err != nil {
		slog.Warn("session not found on rejoin", "sessionId", claims.SessionID, "err", err)
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Get returns details for a specific session identified by the URL parameter.
func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session ID", http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil || claims.SessionID != sessionID {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := h.sessionService.Get(r.Context(), sessionID)
	if err != nil {
		slog.Warn("session not found", "sessionId", sessionID, "err", err)
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// isValidUUID checks whether s is a valid UUID v4 string.
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
