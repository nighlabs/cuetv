package handlers

import (
	"log/slog"
	"net/http"

	"github.com/cuetv/backend/internal/db"
	"github.com/cuetv/backend/internal/middleware"
	"github.com/cuetv/backend/internal/models"
	"github.com/cuetv/backend/internal/services"
	"github.com/go-chi/chi/v5"
)

// PlaybackHandler handles HTTP requests for playback control and video-ended signals.
type PlaybackHandler struct {
	playbackService *services.PlaybackService
	queries         *db.Queries
}

// NewPlaybackHandler creates a new PlaybackHandler with the given playback service and database queries.
func NewPlaybackHandler(playbackService *services.PlaybackService, queries *db.Queries) *PlaybackHandler {
	return &PlaybackHandler{
		playbackService: playbackService,
		queries:         queries,
	}
}

// Command executes a playback command (play, pause, next, prev) for the specified session.
func (h *PlaybackHandler) Command(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session ID", http.StatusBadRequest)
		return
	}

	if !middleware.RequireAdmin(w, r, sessionID) {
		return
	}

	var req models.PlaybackCommandRequest
	if err := decodeJSON(r, &req); err != nil {
		slog.Warn("failed to decode playback command request", "sessionId", sessionID, "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	switch req.Command {
	case "play", "pause", "next", "prev":
		// valid
	default:
		slog.Warn("invalid playback command", "sessionId", sessionID, "command", req.Command)
		http.Error(w, "invalid command", http.StatusBadRequest)
		return
	}

	if err := h.playbackService.Command(r.Context(), sessionID, req.Command); err != nil {
		slog.Error("failed to execute playback command", "sessionId", sessionID, "command", req.Command, "err", err)
		http.Error(w, "failed to execute command", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// VideoEnded handles the signal from a viewer that the current video has finished,
// triggering auto-advance logic to load the next queue item.
func (h *PlaybackHandler) VideoEnded(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	token := r.URL.Query().Get("token")

	if token == "" {
		slog.Warn("video-ended called without viewer token", "sessionId", sessionID)
		http.Error(w, "missing viewer token", http.StatusUnauthorized)
		return
	}

	session, err := h.queries.GetSessionByViewerToken(r.Context(), token)
	if err != nil || session.ID != sessionID {
		slog.Warn("invalid viewer token on video-ended", "sessionId", sessionID)
		http.Error(w, "invalid viewer token", http.StatusUnauthorized)
		return
	}

	if err := h.playbackService.VideoEnded(r.Context(), sessionID); err != nil {
		slog.Error("failed to advance queue on video end", "sessionId", sessionID, "err", err)
		http.Error(w, "failed to advance", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
