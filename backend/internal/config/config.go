package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the backend server. Fields
// include the HTTP Port, JWTSecret for token signing, AdminPortalPassword,
// DatabasePath for SQLite, AdminTokenDuration and FriendTokenDuration for
// JWT expiry, RateLimitPerMinute, TrustedProxies (comma-separated CIDRs),
// SentryDSN for error tracking, and AllowedOrigins for CORS.
type Config struct {
	Port                 string
	JWTSecret            string
	AdminPortalPassword  string
	DatabasePath         string
	AdminTokenDuration   time.Duration
	FriendTokenDuration  time.Duration
	RateLimitPerMinute   int
	TrustedProxies       string
	SentryDSN            string
	AllowedOrigins       string
}

// Load reads configuration from environment variables and returns a validated
// Config. JWT_SECRET and ADMIN_PORTAL_PASSWORD are required; the function
// returns an error if either is missing. All other variables fall back to
// sensible defaults (see getEnvOrDefault calls).
func Load() (*Config, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	adminPassword := os.Getenv("ADMIN_PORTAL_PASSWORD")
	if adminPassword == "" {
		return nil, fmt.Errorf("ADMIN_PORTAL_PASSWORD environment variable is required")
	}

	port := getEnvOrDefault("PORT", "8080")
	dbPath := getEnvOrDefault("DATABASE_PATH", "/data/cuetv.db")

	adminTokenDuration, err := parseDuration("ADMIN_TOKEN_DURATION", "168h")
	if err != nil {
		return nil, fmt.Errorf("invalid ADMIN_TOKEN_DURATION: %w", err)
	}

	friendTokenDuration, err := parseDuration("FRIEND_TOKEN_DURATION", "12h")
	if err != nil {
		return nil, fmt.Errorf("invalid FRIEND_TOKEN_DURATION: %w", err)
	}

	rateLimitPerMinute, err := parseInt("RATE_LIMIT_PER_MINUTE", "10")
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_PER_MINUTE: %w", err)
	}

	return &Config{
		Port:                port,
		JWTSecret:           jwtSecret,
		AdminPortalPassword: adminPassword,
		DatabasePath:        dbPath,
		AdminTokenDuration:  adminTokenDuration,
		FriendTokenDuration: friendTokenDuration,
		RateLimitPerMinute:  rateLimitPerMinute,
		TrustedProxies:      os.Getenv("TRUSTED_PROXIES"),
		SentryDSN:           os.Getenv("SENTRY_DSN"),
		AllowedOrigins:      os.Getenv("ALLOWED_ORIGINS"),
	}, nil
}

// getEnvOrDefault returns the value of the environment variable named key,
// or defaultVal if the variable is empty or unset.
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// parseDuration reads a Go duration string (e.g. "168h") from the environment
// variable key, falling back to defaultVal if unset.
func parseDuration(key, defaultVal string) (time.Duration, error) {
	raw := getEnvOrDefault(key, defaultVal)
	return time.ParseDuration(raw)
}

// parseInt reads an integer from the environment variable key, falling back
// to defaultVal if unset.
func parseInt(key, defaultVal string) (int, error) {
	raw := getEnvOrDefault(key, defaultVal)
	return strconv.Atoi(raw)
}
