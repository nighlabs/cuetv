package models

// CreateSessionRequest is the payload for creating a new session.
// Password is the admin portal password used to authorize session creation.
type CreateSessionRequest struct {
	Password string `json:"password"`
}

// CreateSessionResponse is returned after a new session is created. It contains
// the admin JWT, the human-readable friend key for sharing, and a viewer token
// for the SSE playback stream.
type CreateSessionResponse struct {
	SessionID   string `json:"sessionId"`
	Token       string `json:"token"`
	FriendKey   string `json:"friendKey"`
	ViewerToken string `json:"viewerToken"`
}

// JoinSessionRequest is the payload for joining an existing session as a friend
// using the human-readable friend key (e.g. "happy-tiger-42").
type JoinSessionRequest struct {
	FriendKey string `json:"friendKey"`
}

// JoinSessionResponse is returned when a friend successfully joins a session.
type JoinSessionResponse struct {
	SessionID string `json:"sessionId"`
	Token     string `json:"token"`
}

// RejoinSessionResponse is returned when an admin rejoins an existing session
// using a still-valid JWT. It refreshes the token and returns current session
// details including the friend key and viewer token.
type RejoinSessionResponse struct {
	SessionID   string `json:"sessionId"`
	Token       string `json:"token"`
	FriendKey   string `json:"friendKey"`
	ViewerToken string `json:"viewerToken"`
}

// SessionResponse contains the public details of a session returned by the
// GET session endpoint.
type SessionResponse struct {
	ID          string `json:"id"`
	FriendKey   string `json:"friendKey"`
	ViewerToken string `json:"viewerToken"`
	CreatedAt   string `json:"createdAt"`
}
