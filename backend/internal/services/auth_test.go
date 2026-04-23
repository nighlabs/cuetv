package services

import (
	"testing"
	"time"
)

func newTestAuth() *AuthService {
	return NewAuthService("test-secret", "admin123", time.Hour, 30*time.Minute)
}

func TestVerifyAdminPassword(t *testing.T) {
	auth := newTestAuth()

	if !auth.VerifyAdminPassword("admin123") {
		t.Error("expected correct password to verify")
	}
	if auth.VerifyAdminPassword("wrong") {
		t.Error("expected wrong password to fail")
	}
}

func TestGenerateAndValidateAdminToken(t *testing.T) {
	auth := newTestAuth()

	token, err := auth.GenerateAdminToken("session-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claims.SessionID != "session-123" {
		t.Errorf("expected session-123, got %s", claims.SessionID)
	}
	if claims.Role != "admin" {
		t.Errorf("expected admin role, got %s", claims.Role)
	}
}

func TestGenerateAndValidateFriendToken(t *testing.T) {
	auth := newTestAuth()

	token, err := auth.GenerateFriendToken("session-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claims.SessionID != "session-456" {
		t.Errorf("expected session-456, got %s", claims.SessionID)
	}
	if claims.Role != "friend" {
		t.Errorf("expected friend role, got %s", claims.Role)
	}
}

func TestExpiredToken(t *testing.T) {
	auth := NewAuthService("test-secret", "admin123", -time.Hour, -time.Hour)

	token, err := auth.GenerateAdminToken("session-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = auth.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestInvalidToken(t *testing.T) {
	auth := newTestAuth()

	_, err := auth.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestWrongSecret(t *testing.T) {
	auth1 := NewAuthService("secret-1", "admin123", time.Hour, time.Hour)
	auth2 := NewAuthService("secret-2", "admin123", time.Hour, time.Hour)

	token, err := auth1.GenerateAdminToken("session-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = auth2.ValidateToken(token)
	if err == nil {
		t.Error("expected error when validating with wrong secret")
	}
}
