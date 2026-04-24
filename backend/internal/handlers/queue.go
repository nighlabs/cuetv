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

// QueueHandler handles HTTP requests for queue management operations.
type QueueHandler struct {
	queueService *services.QueueService
	queries      *db.Queries
	validator    middleware.TokenValidator
}

// NewQueueHandler creates a new QueueHandler with the given QueueService.
func NewQueueHandler(queueService *services.QueueService, queries *db.Queries, validator middleware.TokenValidator) *QueueHandler {
	return &QueueHandler{
		queueService: queueService,
		queries:      queries,
		validator:    validator,
	}
}

// List returns all queue items for the specified session.
func (h *QueueHandler) List(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session ID", http.StatusBadRequest)
		return
	}

	// List is read-only — allow both JWT (admin/friend) and viewer token access
	if !authorizeSessionRead(w, r, sessionID, h.queries, h.validator) {
		return
	}

	items, err := h.queueService.List(r.Context(), sessionID)
	if err != nil {
		slog.Error("failed to list queue", "sessionId", sessionID, "err", err)
		http.Error(w, "failed to list queue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// Add adds a new video to the queue for the specified session.
func (h *QueueHandler) Add(w http.ResponseWriter, r *http.Request) {
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

	var req models.AddQueueItemRequest
	if err := decodeJSON(r, &req); err != nil {
		slog.Warn("failed to decode add queue item request", "sessionId", sessionID, "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		slog.Warn("add queue item called without URL", "sessionId", sessionID)
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	if len(req.MarqueeText) > 500 {
		slog.Warn("marquee text exceeds maximum length", "sessionId", sessionID, "length", len(req.MarqueeText))
		http.Error(w, "marquee text must be 500 characters or less", http.StatusBadRequest)
		return
	}

	item, err := h.queueService.Add(r.Context(), sessionID, req.URL, req.MarqueeText)
	if err != nil {
		slog.Warn("failed to add queue item", "sessionId", sessionID, "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

// Reorder updates the ordering of queue items for the specified session.
func (h *QueueHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session ID", http.StatusBadRequest)
		return
	}

	if !middleware.RequireAdmin(w, r, sessionID) {
		return
	}

	var req models.ReorderQueueRequest
	if err := decodeJSON(r, &req); err != nil {
		slog.Warn("failed to decode reorder queue request", "sessionId", sessionID, "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Order) == 0 {
		slog.Warn("reorder queue called with empty order array", "sessionId", sessionID)
		http.Error(w, "order array is required", http.StatusBadRequest)
		return
	}

	if err := h.queueService.Reorder(r.Context(), sessionID, req.Order); err != nil {
		slog.Error("failed to reorder queue", "sessionId", sessionID, "err", err)
		http.Error(w, "failed to reorder queue", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete removes a queue item from the specified session.
func (h *QueueHandler) Delete(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	itemID := chi.URLParam(r, "itemId")
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session ID", http.StatusBadRequest)
		return
	}
	if !isValidUUID(itemID) {
		http.Error(w, "invalid item ID", http.StatusBadRequest)
		return
	}

	if !middleware.RequireAdmin(w, r, sessionID) {
		return
	}

	if err := h.queueService.Delete(r.Context(), sessionID, itemID); err != nil {
		slog.Error("failed to delete queue item", "sessionId", sessionID, "itemId", itemID, "err", err)
		http.Error(w, "failed to delete item", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateMarqueeText updates the marquee text for a single queue item.
func (h *QueueHandler) UpdateMarqueeText(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	itemID := chi.URLParam(r, "itemId")
	if !isValidUUID(sessionID) || !isValidUUID(itemID) {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}

	claims := middleware.GetClaims(r)
	if claims == nil || claims.SessionID != sessionID {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.UpdateMarqueeTextRequest
	if err := decodeJSON(r, &req); err != nil {
		slog.Warn("failed to decode update marquee text request", "sessionId", sessionID, "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.MarqueeText) > 500 {
		slog.Warn("marquee text exceeds maximum length", "sessionId", sessionID, "length", len(req.MarqueeText))
		http.Error(w, "marquee text must be 500 characters or less", http.StatusBadRequest)
		return
	}

	if err := h.queueService.UpdateMarqueeText(r.Context(), sessionID, itemID, req.MarqueeText); err != nil {
		slog.Error("failed to update marquee text", "sessionId", sessionID, "itemId", itemID, "err", err)
		http.Error(w, "failed to update marquee text", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
