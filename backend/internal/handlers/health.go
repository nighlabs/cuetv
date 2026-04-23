package handlers

import (
	"encoding/json"
	"net/http"
)

// decodeJSON reads and decodes the request body into dst, rejecting requests
// that contain unknown fields. This prevents clients from silently sending
// extra data that could indicate probing or misconfiguration.
func decodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// Health handles the health check endpoint, returning a JSON status response.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
