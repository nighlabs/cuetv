package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_RequiresJWTSecret(t *testing.T) {
	clearEnv(t)
	os.Setenv("ADMIN_PORTAL_PASSWORD", "test")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is missing")
	}
}

func TestLoad_RequiresAdminPortalPassword(t *testing.T) {
	clearEnv(t)
	os.Setenv("JWT_SECRET", "secret")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when ADMIN_PORTAL_PASSWORD is missing")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	os.Setenv("JWT_SECRET", "secret")
	os.Setenv("ADMIN_PORTAL_PASSWORD", "password")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected port 8080, got %s", cfg.Port)
	}
	if cfg.DatabasePath != "/data/cuetv.db" {
		t.Errorf("expected default db path, got %s", cfg.DatabasePath)
	}
	if cfg.AdminTokenDuration != 168*time.Hour {
		t.Errorf("expected 168h admin token duration, got %v", cfg.AdminTokenDuration)
	}
	if cfg.FriendTokenDuration != 12*time.Hour {
		t.Errorf("expected 12h friend token duration, got %v", cfg.FriendTokenDuration)
	}
	if cfg.RateLimitPerMinute != 10 {
		t.Errorf("expected rate limit 10, got %d", cfg.RateLimitPerMinute)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	clearEnv(t)
	os.Setenv("JWT_SECRET", "secret")
	os.Setenv("ADMIN_PORTAL_PASSWORD", "password")
	os.Setenv("PORT", "9090")
	os.Setenv("DATABASE_PATH", "/tmp/test.db")
	os.Setenv("ADMIN_TOKEN_DURATION", "24h")
	os.Setenv("FRIEND_TOKEN_DURATION", "6h")
	os.Setenv("RATE_LIMIT_PER_MINUTE", "20")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("expected /tmp/test.db, got %s", cfg.DatabasePath)
	}
	if cfg.AdminTokenDuration != 24*time.Hour {
		t.Errorf("expected 24h, got %v", cfg.AdminTokenDuration)
	}
	if cfg.RateLimitPerMinute != 20 {
		t.Errorf("expected 20, got %d", cfg.RateLimitPerMinute)
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"JWT_SECRET", "ADMIN_PORTAL_PASSWORD", "PORT", "DATABASE_PATH",
		"ADMIN_TOKEN_DURATION", "FRIEND_TOKEN_DURATION", "RATE_LIMIT_PER_MINUTE",
		"TRUSTED_PROXIES", "SENTRY_DSN", "ALLOWED_ORIGINS",
	} {
		os.Unsetenv(key)
	}
}
