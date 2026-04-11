package services

import (
	"crypto/subtle"
	"fmt"
	"time"

	"github.com/cuetv/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// AuthService handles JWT token generation and validation for admin and friend
// roles, as well as admin password verification. It holds the shared JWT signing
// secret and per-role token durations.
type AuthService struct {
	jwtSecret           []byte
	adminPassword       string
	adminTokenDuration  time.Duration
	friendTokenDuration time.Duration
}

// jwtClaims extends the standard JWT registered claims with CueTV-specific
// fields: the session ID the token is scoped to and the role (admin or friend).
type jwtClaims struct {
	SessionID string `json:"sessionId"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// NewAuthService creates an AuthService with the given JWT signing secret,
// admin portal password, and per-role token durations. The jwtSecret and
// adminPassword are sourced from environment variables at startup.
func NewAuthService(jwtSecret, adminPassword string, adminDuration, friendDuration time.Duration) *AuthService {
	return &AuthService{
		jwtSecret:           []byte(jwtSecret),
		adminPassword:       adminPassword,
		adminTokenDuration:  adminDuration,
		friendTokenDuration: friendDuration,
	}
}

// VerifyAdminPassword checks whether the supplied password matches the
// configured admin portal password. It uses crypto/subtle.ConstantTimeCompare
// to prevent timing side-channel attacks — the comparison takes the same amount
// of time regardless of how many bytes match.
func (s *AuthService) VerifyAdminPassword(password string) bool {
	return subtle.ConstantTimeCompare([]byte(password), []byte(s.adminPassword)) == 1
}

// GenerateAdminToken creates a signed JWT for the admin role, scoped to the
// given session ID. The token is valid for the configured admin token duration.
func (s *AuthService) GenerateAdminToken(sessionID string) (string, error) {
	return s.generateToken(sessionID, "admin", s.adminTokenDuration)
}

// GenerateFriendToken creates a signed JWT for the friend role, scoped to the
// given session ID. Friend tokens have a shorter lifetime than admin tokens.
func (s *AuthService) GenerateFriendToken(sessionID string) (string, error) {
	return s.generateToken(sessionID, "friend", s.friendTokenDuration)
}

// ValidateToken parses and validates a JWT string, returning the extracted
// claims (session ID and role) on success. The signing method is explicitly
// checked to be HMAC before trusting the token — this prevents an attacker
// from switching the algorithm to "none" or an asymmetric method to bypass
// signature verification.
func (s *AuthService) ValidateToken(tokenString string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &models.Claims{
		SessionID: claims.SessionID,
		Role:      claims.Role,
	}, nil
}

// generateToken is the internal helper that creates and signs a JWT with the
// given session ID, role, and expiry duration. All exported token generators
// delegate to this method.
func (s *AuthService) generateToken(sessionID, role string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		SessionID: sessionID,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
