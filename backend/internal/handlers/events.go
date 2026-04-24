package handlers

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cuetv/backend/internal/broker"
	"github.com/cuetv/backend/internal/db"
	"github.com/go-chi/chi/v5"
)

// EventsHandler handles SSE streaming connections for viewer playback events.
type EventsHandler struct {
	sseBroker *broker.SSEBroker
	queries   *db.Queries
}

// NewEventsHandler creates a new EventsHandler with the given SSE broker and database queries.
func NewEventsHandler(sseBroker *broker.SSEBroker, queries *db.Queries) *EventsHandler {
	return &EventsHandler{
		sseBroker: sseBroker,
		queries:   queries,
	}
}

// SSEStream establishes a Server-Sent Events connection for a viewer,
// validating the viewer token and streaming playback commands for the session.
func (h *EventsHandler) SSEStream(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if !isValidUUID(sessionID) {
		http.Error(w, "invalid session ID", http.StatusBadRequest)
		return
	}

	token := r.URL.Query().Get("token")

	if token == "" {
		slog.Warn("SSE connection attempted without viewer token", "sessionId", sessionID)
		http.Error(w, "missing viewer token", http.StatusUnauthorized)
		return
	}

	// Validate viewer token belongs to this session
	session, err := h.queries.GetSessionByViewerToken(r.Context(), token)
	if err != nil || session.ID != sessionID {
		slog.Warn("invalid viewer token on SSE connection", "sessionId", sessionID)
		http.Error(w, "invalid viewer token", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("streaming not supported by response writer", "sessionId", sessionID)
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := h.sseBroker.Subscribe(sessionID)
	defer h.sseBroker.Unsubscribe(sessionID, ch)

	// Send initial connection event
	w.Write([]byte(broker.FormatSSEEvent("connected")))
	flusher.Flush()

	// Send the currently-playing video so late-joining viewers catch up
	// immediately instead of waiting for the next admin command.
	currentIndex, err := h.queries.GetCurrentIndex(r.Context(), sessionID)
	if err == nil {
		item, err := h.queries.GetQueueItemAtPosition(r.Context(), db.GetQueueItemAtPositionParams{
			SessionID: sessionID,
			Position:  currentIndex,
		})
		if err == nil {
			w.Write([]byte(broker.FormatSSEEvent(fmt.Sprintf("load:%s", item.YoutubeVideoID))))
			flusher.Flush()
		} else if err != sql.ErrNoRows {
			slog.Warn("failed to get current queue item for SSE catchup", "sessionId", sessionID, "err", err)
		}
	}

	for {
		select {
		case <-r.Context().Done():
			slog.Debug("SSE client disconnected", "sessionId", sessionID)
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			w.Write([]byte(broker.FormatSSEEvent(event)))
			flusher.Flush()
		}
	}
}
