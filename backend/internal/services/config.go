package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cuetv/backend/internal/broker"
	"github.com/cuetv/backend/internal/db"
	"github.com/cuetv/backend/internal/models"
)

// ConfigService manages the room configuration for a session, including
// marquee display settings. Changes are broadcast to connected admins via
// the WebSocket hub.
type ConfigService struct {
	database *sql.DB
	queries  *db.Queries
	wsHub    *broker.WSHub
}

// NewConfigService creates a ConfigService with the given database connection,
// sqlc queries, and WebSocket hub for broadcasting config changes.
func NewConfigService(database *sql.DB, queries *db.Queries, wsHub *broker.WSHub) *ConfigService {
	return &ConfigService{database: database, queries: queries, wsHub: wsHub}
}

// Get retrieves the current room configuration for a session.
func (s *ConfigService) Get(ctx context.Context, sessionID string) (*models.RoomConfigResponse, error) {
	cfg, err := s.queries.GetRoomConfig(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("getting room config: %w", err)
	}

	return &models.RoomConfigResponse{
		TopMarqueeEnabled:    cfg.TopMarqueeEnabled != 0,
		BottomMarqueeEnabled: cfg.BottomMarqueeEnabled != 0,
		TopMarqueeLabel:      cfg.TopMarqueeLabel,
		BottomMarqueeLabel:   cfg.BottomMarqueeLabel,
		TopMarqueeSource:     cfg.TopMarqueeSource,
		BottomMarqueeSource:  cfg.BottomMarqueeSource,
		CurrentIndex:         int(cfg.CurrentIndex),
	}, nil
}

// Update applies a partial update to the room configuration using a
// read-modify-write pattern inside a transaction. The current config is read
// first, then only the fields present in the request (non-nil pointer fields)
// are overwritten. This allows clients to send partial updates — e.g. toggling
// a single marquee — without needing to re-send the entire config. The
// transaction protects against concurrent partial updates: two PATCH requests
// arriving simultaneously could otherwise cause one update to be silently lost.
// After persisting the changes, a config:updated event is broadcast to all
// connected admins.
func (s *ConfigService) Update(ctx context.Context, sessionID string, req models.UpdateRoomConfigRequest) (*models.RoomConfigResponse, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	current, err := qtx.GetRoomConfig(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("getting current config: %w", err)
	}

	params := db.UpdateRoomConfigParams{
		SessionID:            sessionID,
		TopMarqueeEnabled:    current.TopMarqueeEnabled,
		BottomMarqueeEnabled: current.BottomMarqueeEnabled,
		TopMarqueeLabel:      current.TopMarqueeLabel,
		BottomMarqueeLabel:   current.BottomMarqueeLabel,
		TopMarqueeSource:     current.TopMarqueeSource,
		BottomMarqueeSource:  current.BottomMarqueeSource,
	}

	if req.TopMarqueeEnabled != nil {
		params.TopMarqueeEnabled = boolToInt(*req.TopMarqueeEnabled)
	}
	if req.BottomMarqueeEnabled != nil {
		params.BottomMarqueeEnabled = boolToInt(*req.BottomMarqueeEnabled)
	}
	if req.TopMarqueeLabel != nil {
		cleaned := stripHTMLTags(*req.TopMarqueeLabel)
		params.TopMarqueeLabel = cleaned
	}
	if req.BottomMarqueeLabel != nil {
		cleaned := stripHTMLTags(*req.BottomMarqueeLabel)
		params.BottomMarqueeLabel = cleaned
	}
	if req.TopMarqueeSource != nil {
		params.TopMarqueeSource = *req.TopMarqueeSource
	}
	if req.BottomMarqueeSource != nil {
		params.BottomMarqueeSource = *req.BottomMarqueeSource
	}

	if err := qtx.UpdateRoomConfig(ctx, params); err != nil {
		return nil, fmt.Errorf("updating room config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing config update: %w", err)
	}

	slog.Info("room config updated", "sessionId", sessionID)
	s.broadcastConfigUpdated(sessionID)

	return s.Get(ctx, sessionID)
}

// broadcastConfigUpdated sends a config:updated WebSocket message to all admins
// connected to the given session, prompting their UIs to refetch the config.
func (s *ConfigService) broadcastConfigUpdated(sessionID string) {
	msg, _ := json.Marshal(map[string]string{"type": "config:updated"})
	s.wsHub.Broadcast(sessionID, msg)
}

// boolToInt converts a boolean to an int64 for SQLite storage. SQLite does not
// have a native boolean type, so booleans are stored as 0 (false) or 1 (true).
func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
