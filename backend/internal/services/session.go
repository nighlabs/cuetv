package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/cuetv/backend/internal/db"
	"github.com/cuetv/backend/internal/models"
	"github.com/google/uuid"
)

// SessionService manages the lifecycle of CueTV sessions — creation, joining
// via friend key, rejoining as admin, and retrieval. It coordinates the
// AuthService for token issuance and the database for persistence.
type SessionService struct {
	queries     *db.Queries
	authService *AuthService
	database    *sql.DB
}

// NewSessionService creates a SessionService with the given database connection,
// sqlc queries, and auth service for token generation.
func NewSessionService(database *sql.DB, queries *db.Queries, authService *AuthService) *SessionService {
	return &SessionService{
		queries:     queries,
		authService: authService,
		database:    database,
	}
}

// Create initializes a new session. It generates a unique friend key and a
// cryptographic viewer token, then inserts both the session and its default
// room config inside a single database transaction. This ensures the session
// and config are created atomically — if either insert fails, neither is
// persisted. Returns an admin JWT, the friend key, and the viewer token.
func (s *SessionService) Create(ctx context.Context) (*models.CreateSessionResponse, error) {
	sessionID := uuid.New().String()
	viewerToken, err := generateViewerToken()
	if err != nil {
		return nil, fmt.Errorf("generating viewer token: %w", err)
	}

	checker := &dbFriendKeyChecker{queries: s.queries, ctx: ctx}
	friendKey, err := GenerateFriendKey(checker)
	if err != nil {
		return nil, fmt.Errorf("generating friend key: %w", err)
	}

	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	err = qtx.CreateSession(ctx, db.CreateSessionParams{
		ID:            sessionID,
		FriendJoinKey: friendKey,
		ViewerToken:   viewerToken,
	})
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	err = qtx.CreateRoomConfig(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("creating room config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing session creation: %w", err)
	}

	token, err := s.authService.GenerateAdminToken(sessionID)
	if err != nil {
		return nil, fmt.Errorf("generating admin token: %w", err)
	}

	slog.Info("session created", "sessionId", sessionID, "friendKey", friendKey)

	return &models.CreateSessionResponse{
		SessionID:   sessionID,
		Token:       token,
		FriendKey:   friendKey,
		ViewerToken: viewerToken,
	}, nil
}

// Join looks up an existing session by friend key and issues a friend-role JWT.
// The friend key is normalized to lowercase before lookup, making join codes
// case-insensitive so users can share them verbally without worrying about
// capitalization.
func (s *SessionService) Join(ctx context.Context, friendKey string) (*models.JoinSessionResponse, error) {
	normalized := NormalizeFriendKey(friendKey)

	session, err := s.queries.GetSessionByFriendKey(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("session not found for friend key: %w", err)
	}

	token, err := s.authService.GenerateFriendToken(session.ID)
	if err != nil {
		return nil, fmt.Errorf("generating friend token: %w", err)
	}

	slog.Info("friend joined session", "sessionId", session.ID)

	return &models.JoinSessionResponse{
		SessionID: session.ID,
		Token:     token,
	}, nil
}

// Rejoin allows an admin to reconnect to an existing session and receive a
// fresh admin JWT. This is used when an admin's previous token has expired or
// been lost — a new token is issued each time rather than reusing old ones.
func (s *SessionService) Rejoin(ctx context.Context, sessionID string) (*models.RejoinSessionResponse, error) {
	session, err := s.queries.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	token, err := s.authService.GenerateAdminToken(session.ID)
	if err != nil {
		return nil, fmt.Errorf("generating admin token: %w", err)
	}

	slog.Info("admin rejoined session", "sessionId", session.ID)

	return &models.RejoinSessionResponse{
		SessionID:   session.ID,
		Token:       token,
		FriendKey:   session.FriendJoinKey,
		ViewerToken: session.ViewerToken,
	}, nil
}

// Get retrieves the public details of a session by its ID.
func (s *SessionService) Get(ctx context.Context, sessionID string) (*models.SessionResponse, error) {
	session, err := s.queries.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	return &models.SessionResponse{
		ID:          session.ID,
		FriendKey:   session.FriendJoinKey,
		ViewerToken: session.ViewerToken,
		CreatedAt:   session.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// generateViewerToken creates a cryptographically random viewer token by
// reading 32 bytes (256 bits) from crypto/rand and hex-encoding them. 256 bits
// provides sufficient entropy to make brute-force guessing infeasible.
func generateViewerToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// dbFriendKeyChecker adapts the database queries to satisfy the FriendKeyChecker
// interface, allowing GenerateFriendKey to check for collisions against the
// database without directly depending on the db package.
type dbFriendKeyChecker struct {
	queries *db.Queries
	ctx     context.Context
}

func (c *dbFriendKeyChecker) FriendKeyExists(key string) (bool, error) {
	return c.queries.FriendKeyExists(c.ctx, key)
}
