package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cuetv/backend/internal/models"
)

type contextKey string

// ClaimsKey is the context key used to store and retrieve authenticated
// JWT claims from an http.Request's context.
const ClaimsKey contextKey = "claims"

// TokenValidator is the interface required by the Auth middleware to verify
// JWT tokens. Implementations are responsible for signature verification,
// expiry checks, and claims extraction.
type TokenValidator interface {
	ValidateToken(tokenString string) (*models.Claims, error)
}

// Auth returns middleware that extracts a Bearer token from the Authorization
// header, validates it using the provided TokenValidator, and injects the
// resulting claims into the request context under ClaimsKey. Requests without
// a valid Bearer token receive a 401 Unauthorized response.
func Auth(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}

			claims, err := validator.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims extracts the authenticated claims from the request context.
// It returns nil if the Auth middleware has not run or the token was invalid.
func GetClaims(r *http.Request) *models.Claims {
	claims, _ := r.Context().Value(ClaimsKey).(*models.Claims)
	return claims
}

// RequireAdmin checks that the authenticated user has the "admin" role
// and that their JWT session matches the URL's session ID. Returns false
// and writes a 403 response if the check fails.
func RequireAdmin(w http.ResponseWriter, r *http.Request, sessionID string) bool {
	claims := GetClaims(r)
	if claims == nil || claims.SessionID != sessionID {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	if claims.Role != "admin" {
		slog.Warn("non-admin attempted admin operation", "sessionId", sessionID, "role", claims.Role)
		http.Error(w, "forbidden: admin role required", http.StatusForbidden)
		return false
	}
	return true
}
