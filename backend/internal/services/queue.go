package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/cuetv/backend/internal/broker"
	"github.com/cuetv/backend/internal/db"
	"github.com/cuetv/backend/internal/models"
	"github.com/google/uuid"
)

// htmlTagRegex matches HTML/XML tags for stripping from user input.
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// stripHTMLTags removes all HTML/XML tags from the input string as a
// defense-in-depth measure against XSS. React escapes by default, but
// server-side stripping prevents issues if rendering changes.
func stripHTMLTags(s string) string {
	return htmlTagRegex.ReplaceAllString(s, "")
}

// QueueService manages the video queue for a session — adding, listing,
// reordering, and deleting queue items. It broadcasts changes to connected
// admin clients via the WebSocket hub so all admin UIs stay in sync.
type QueueService struct {
	queries  *db.Queries
	database *sql.DB
	wsHub    *broker.WSHub
}

// NewQueueService creates a QueueService with the given database connection,
// sqlc queries, and WebSocket hub for broadcasting queue changes.
func NewQueueService(database *sql.DB, queries *db.Queries, wsHub *broker.WSHub) *QueueService {
	return &QueueService{
		queries:  queries,
		database: database,
		wsHub:    wsHub,
	}
}

// Add validates the submitted YouTube URL server-side, extracts the video ID,
// and appends a new item to the end of the session's queue. The position is
// assigned as the current item count, placing the new item last. After
// insertion, a queue:updated event is broadcast to all connected admins.
func (s *QueueService) Add(ctx context.Context, sessionID, rawURL, marqueeText string) (*models.QueueItemResponse, error) {
	videoID, err := ExtractYouTubeVideoID(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid YouTube URL: %w", err)
	}

	marqueeText = stripHTMLTags(marqueeText)

	count, err := s.queries.GetQueueItemCount(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("getting queue count: %w", err)
	}

	itemID := uuid.New().String()
	err = s.queries.AddQueueItem(ctx, db.AddQueueItemParams{
		ID:             itemID,
		SessionID:      sessionID,
		YoutubeVideoID: videoID,
		YoutubeUrl:     rawURL,
		MarqueeText:    marqueeText,
		Position:       count,
	})
	if err != nil {
		return nil, fmt.Errorf("adding queue item: %w", err)
	}

	slog.Info("queue item added", "sessionId", sessionID, "itemId", itemID, "videoId", videoID)
	s.broadcastQueueUpdated(sessionID)

	return &models.QueueItemResponse{
		ID:             itemID,
		YouTubeVideoID: videoID,
		YouTubeURL:     rawURL,
		MarqueeText:    marqueeText,
		Position:       int(count),
	}, nil
}

// List returns all queue items for a session, ordered by position.
func (s *QueueService) List(ctx context.Context, sessionID string) ([]models.QueueItemResponse, error) {
	items, err := s.queries.GetQueueItems(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("listing queue items: %w", err)
	}

	result := make([]models.QueueItemResponse, len(items))
	for i, item := range items {
		result[i] = models.QueueItemResponse{
			ID:             item.ID,
			YouTubeVideoID: item.YoutubeVideoID,
			YouTubeURL:     item.YoutubeUrl,
			MarqueeText:    item.MarqueeText,
			Position:       int(item.Position),
			AddedAt:        item.AddedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return result, nil
}

// Reorder replaces the positions of all queue items in a single transaction.
// The caller provides the complete ordered list of item IDs — each item's
// position is set to its index in the array. A full-array reorder (rather than
// partial swaps) is used because it is simpler to reason about, avoids edge
// cases with gaps or duplicates, and the queue size is small enough that
// updating every row is not a performance concern.
func (s *QueueService) Reorder(ctx context.Context, sessionID string, order []string) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	for i, itemID := range order {
		err := qtx.UpdateQueueItemPosition(ctx, db.UpdateQueueItemPositionParams{
			Position:  int64(i),
			ID:        itemID,
			SessionID: sessionID,
		})
		if err != nil {
			return fmt.Errorf("updating position for item %s: %w", itemID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing reorder: %w", err)
	}

	slog.Info("queue reordered", "sessionId", sessionID, "itemCount", len(order))
	s.broadcastQueueUpdated(sessionID)
	return nil
}

// Delete removes a queue item from a session by item ID and broadcasts a
// queue:updated event to all connected admins.
func (s *QueueService) Delete(ctx context.Context, sessionID, itemID string) error {
	err := s.queries.DeleteQueueItem(ctx, db.DeleteQueueItemParams{
		ID:        itemID,
		SessionID: sessionID,
	})
	if err != nil {
		return fmt.Errorf("deleting queue item: %w", err)
	}

	slog.Info("queue item deleted", "sessionId", sessionID, "itemId", itemID)
	s.broadcastQueueUpdated(sessionID)
	return nil
}

// broadcastQueueUpdated sends a queue:updated WebSocket message to all admins
// connected to the given session, prompting their UIs to refetch the queue.
func (s *QueueService) broadcastQueueUpdated(sessionID string) {
	msg, _ := json.Marshal(map[string]string{"type": "queue:updated"})
	s.wsHub.Broadcast(sessionID, msg)
}
