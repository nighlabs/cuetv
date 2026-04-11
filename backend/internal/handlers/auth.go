package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/cuetv/backend/internal/models"
	"github.com/cuetv/backend/internal/services"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new AuthHandler with the given AuthService.
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Verify checks whether the provided admin password is valid and returns the result.
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req models.AdminVerifyRequest
	if err := decodeJSON(r, &req); err != nil {
		slog.Warn("failed to decode admin verify request", "err", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	valid := h.authService.VerifyAdminPassword(req.Password)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AdminVerifyResponse{Valid: valid})
}
