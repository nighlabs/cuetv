package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/cuetv/backend/internal/db"
	"github.com/cuetv/backend/internal/middleware"
	"github.com/cuetv/backend/internal/models"
	"github.com/cuetv/backend/internal/services"
	"github.com/go-chi/chi/v5"
)

// ConfigHandler handles HTTP requests for session room configuration.
type ConfigHandler struct {
	configService *services.ConfigService
	queries       *db.Queries
	validator     middleware.TokenValidator
}

// NewConfigHandler creates a new ConfigHandler with the given ConfigService.
func NewConfigHandler(configService *services.ConfigService, queries *db.Queries, validator middleware.TokenValidator) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
		queries:       queries,
		validator:     validator,
	}
}

// Get returns the room configuration for the specified session.
func (h *ConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session ID", http.StatusBadRequest)
		return
	}

	// Get is read-only — allow both JWT (admin/friend) and viewer token access
	if !authorizeSessionRead(w, r, sessionID, h.queries, h.validator) {
		return
	}

	cfg, err := h.configService.Get(r.Context(), sessionID)
	if err != nil {
		slog.Error("failed to get config", "sessionId", sessionID, "err", err)
		http.Error(w, "failed to get config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// Update modifies the room configuration for the specified session.
func (h *ConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session ID", http.StatusBadRequest)
		return
	}

	if !middleware.RequireAdmin(w, r, sessionID) {
		return
	}

	var req models.UpdateRoomConfigRequest
	if err := decodeJSON(r, &req); err != nil {
		slog.Warn("failed to decode update config request", "sessionId", sessionID, "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.TopMarqueeLabel != nil && len(*req.TopMarqueeLabel) > 50 {
		slog.Warn("top marquee label exceeds maximum length", "sessionId", sessionID, "length", len(*req.TopMarqueeLabel))
		http.Error(w, "marquee label must be 50 characters or less", http.StatusBadRequest)
		return
	}
	if req.BottomMarqueeLabel != nil && len(*req.BottomMarqueeLabel) > 50 {
		slog.Warn("bottom marquee label exceeds maximum length", "sessionId", sessionID, "length", len(*req.BottomMarqueeLabel))
		http.Error(w, "marquee label must be 50 characters or less", http.StatusBadRequest)
		return
	}

	cfg, err := h.configService.Update(r.Context(), sessionID, req)
	if err != nil {
		slog.Error("failed to update config", "sessionId", sessionID, "err", err)
		http.Error(w, "failed to update config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}
