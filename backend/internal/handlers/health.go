package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cuetv/backend/internal/db"
	"github.com/cuetv/backend/internal/middleware"
)

// decodeJSON reads and decodes the request body into dst, rejecting requests
// that contain unknown fields. This prevents clients from silently sending
// extra data that could indicate probing or misconfiguration.
func decodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// authorizeSessionRead checks whether the request has read access to the given
// session, via either a valid JWT (admin/friend) or a viewer token query param.
// It handles both cases where the Auth middleware has run (claims in context) and
// where it hasn't (JWT parsed directly from the Authorization header). This allows
// these endpoints to live outside the Auth middleware group while still accepting
// JWT auth from admin/friend clients.
// Returns true if authorized. Writes an HTTP error and returns false if not.
func authorizeSessionRead(w http.ResponseWriter, r *http.Request, sessionID string, queries *db.Queries, validator middleware.TokenValidator) bool {
	// Check if Auth middleware already injected claims
	claims := middleware.GetClaims(r)
	if claims != nil && claims.SessionID == sessionID {
		return true
	}

	// Try parsing JWT directly from Authorization header (for requests
	// outside the Auth middleware group, e.g. admin page fetching queue)
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr != authHeader {
			if parsed, err := validator.ValidateToken(tokenStr); err == nil && parsed.SessionID == sessionID {
				return true
			}
		}
	}

	// Fall back to viewer token in query param (viewer page)
	token := r.URL.Query().Get("token")
	if token != "" {
		session, err := queries.GetSessionByViewerToken(r.Context(), token)
		if err == nil && session.ID == sessionID {
			return true
		}
	}

	http.Error(w, "unauthorized", http.StatusUnauthorized)
	return false
}

// Health handles the health check endpoint, returning a JSON status response.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
