package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cuetv/backend/internal/broker"
	"github.com/cuetv/backend/internal/db"
)

// PlaybackService handles playback commands (play, pause, next, prev) and
// auto-advance logic when a video ends. It maintains a per-session debounce map
// to prevent duplicate auto-advance signals from multiple viewers watching the
// same session — when several viewers finish a video at roughly the same time,
// only the first signal triggers an advance.
type PlaybackService struct {
	queries   *db.Queries
	sseBroker *broker.SSEBroker
	wsHub     *broker.WSHub

	mu            sync.Mutex
	lastAdvance   map[string]time.Time // sessionID -> last auto-advance time
	debounceDur   time.Duration
}

// NewPlaybackService creates a PlaybackService with the given sqlc queries,
// SSE broker for viewer commands, and WebSocket hub for admin notifications.
func NewPlaybackService(queries *db.Queries, sseBroker *broker.SSEBroker, wsHub *broker.WSHub) *PlaybackService {
	return &PlaybackService{
		queries:     queries,
		sseBroker:   sseBroker,
		wsHub:       wsHub,
		lastAdvance: make(map[string]time.Time),
		debounceDur: 3 * time.Second,
	}
}

// Command executes a playback command for the given session. "play" and "pause"
// are broadcast directly to viewers via SSE. "next" and "prev" advance the
// queue position by +1 or -1 respectively.
func (s *PlaybackService) Command(ctx context.Context, sessionID, command string) error {
	switch command {
	case "play":
		// Load the current video if this is the first play command for the
		// session — viewers need a video ID before playVideo() can succeed.
		// Subsequent play commands just resume the already-loaded video.
		return s.playOrLoad(ctx, sessionID)
	case "pause":
		s.sseBroker.Broadcast(sessionID, "pause")
	case "next":
		return s.advance(ctx, sessionID, 1)
	case "prev":
		return s.advance(ctx, sessionID, -1)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	slog.Debug("playback command broadcast", "sessionId", sessionID, "command", command)
	return nil
}

// VideoEnded handles the video-ended signal from a viewer. It debounces
// auto-advance with a 3-second window per session. The 3-second duration
// accounts for slight timing differences between multiple viewers finishing the
// same video — without debouncing, each viewer's end signal would advance the
// queue, skipping videos.
func (s *PlaybackService) VideoEnded(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	last, exists := s.lastAdvance[sessionID]
	now := time.Now()
	if exists && now.Sub(last) < s.debounceDur {
		s.mu.Unlock()
		slog.Debug("auto-advance debounced", "sessionId", sessionID)
		return nil
	}
	s.lastAdvance[sessionID] = now
	s.mu.Unlock()

	return s.advance(ctx, sessionID, 1)
}

// playOrLoad sends the current video to viewers. If a video is at the current
// queue position, it broadcasts a load:{videoId} event so the player loads it.
// If no video is at the current position (empty queue), it broadcasts a plain
// "play" event as a resume signal.
func (s *PlaybackService) playOrLoad(ctx context.Context, sessionID string) error {
	currentIndex, err := s.queries.GetCurrentIndex(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("getting current index: %w", err)
	}

	item, err := s.queries.GetQueueItemAtPosition(ctx, db.GetQueueItemAtPositionParams{
		SessionID: sessionID,
		Position:  currentIndex,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			// No video at current position — just send a resume signal
			s.sseBroker.Broadcast(sessionID, "play")
			slog.Debug("play command broadcast (no video loaded)", "sessionId", sessionID)
			return nil
		}
		return fmt.Errorf("getting current queue item: %w", err)
	}

	// Video exists at current position — load it on all viewers
	s.sseBroker.Broadcast(sessionID, fmt.Sprintf("load:%s", item.YoutubeVideoID))
	s.broadcastQueueUpdated(sessionID)
	s.broadcastConfigUpdated(sessionID)

	slog.Debug("play command broadcast with video load", "sessionId", sessionID, "videoId", item.YoutubeVideoID)
	return nil
}

// advance moves the queue position by delta (typically +1 for next, -1 for
// prev). A delta of +1 advances forward; -1 goes back. The position is clamped
// to a minimum of 0. If no queue item exists at the new position because the
// queue has ended (sql.ErrNoRows), a queue:ended event is broadcast to viewers.
// Any other database error is returned to the caller.
func (s *PlaybackService) advance(ctx context.Context, sessionID string, delta int) error {
	currentIndex, err := s.queries.GetCurrentIndex(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("getting current index: %w", err)
	}

	newIndex := currentIndex + int64(delta)
	if newIndex < 0 {
		newIndex = 0
	}

	// Try to get item at new position; distinguish "no rows" (end of queue)
	// from real database errors.
	item, err := s.queries.GetQueueItemAtPosition(ctx, db.GetQueueItemAtPositionParams{
		SessionID: sessionID,
		Position:  newIndex,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			// No item at this position — queue has ended
			s.sseBroker.Broadcast(sessionID, "queue:ended")
			slog.Info("queue ended", "sessionId", sessionID)
			return nil
		}
		return fmt.Errorf("getting queue item at position %d: %w", newIndex, err)
	}

	// Update current index
	err = s.queries.UpdateCurrentIndex(ctx, db.UpdateCurrentIndexParams{
		CurrentIndex: newIndex,
		SessionID:    sessionID,
	})
	if err != nil {
		return fmt.Errorf("updating current index: %w", err)
	}

	// Broadcast to viewers via SSE
	s.sseBroker.Broadcast(sessionID, fmt.Sprintf("load:%s", item.YoutubeVideoID))

	// Broadcast to admins via WebSocket — both queue and config changed
	// since currentIndex moved to a new position
	s.broadcastQueueUpdated(sessionID)
	s.broadcastConfigUpdated(sessionID)

	slog.Info("playback advanced",
		"sessionId", sessionID,
		"newIndex", newIndex,
		"videoId", item.YoutubeVideoID,
		"viewerCount", s.sseBroker.SubscriberCount(sessionID),
	)

	return nil
}

// broadcastQueueUpdated sends a queue:updated WebSocket message to all admins
// connected to the given session.
func (s *PlaybackService) broadcastQueueUpdated(sessionID string) {
	msg := []byte(`{"type":"queue:updated"}`)
	s.wsHub.Broadcast(sessionID, msg)
}

// broadcastConfigUpdated sends a config:updated WebSocket message so admins
// refetch room config (e.g. when currentIndex changes on next/prev).
func (s *PlaybackService) broadcastConfigUpdated(sessionID string) {
	msg := []byte(`{"type":"config:updated"}`)
	s.wsHub.Broadcast(sessionID, msg)
}
