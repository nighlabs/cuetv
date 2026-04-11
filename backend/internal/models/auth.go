package models

// AdminVerifyRequest is the payload for verifying the admin portal password.
type AdminVerifyRequest struct {
	Password string `json:"password"`
}

// AdminVerifyResponse indicates whether the submitted admin password was correct.
type AdminVerifyResponse struct {
	Valid bool `json:"valid"`
}

// Claims holds the JWT payload for authenticated users. Role is either
// "admin" (full session control) or "friend" (queue contributions only).
type Claims struct {
	SessionID string `json:"sessionId"`
	Role      string `json:"role"` // "admin" or "friend"
}
