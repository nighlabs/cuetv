package middleware

import (
	"net/http"
	"strings"
)

// CORS returns middleware that enforces an explicit origin allowlist for
// Cross-Origin Resource Sharing. Wildcard origins are never permitted.
// When the request origin matches an allowed entry, the middleware sets
// Access-Control-Allow-Origin, Allow-Methods, Allow-Headers, Allow-Credentials,
// and Max-Age headers. It also unconditionally adds X-Content-Type-Options and
// Strict-Transport-Security security headers. Preflight OPTIONS requests are
// handled with a 204 No Content response.
func CORS(allowedOrigins string) func(http.Handler) http.Handler {
	origins := parseOrigins(allowedOrigins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if isAllowed(origin, origins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseOrigins splits a comma-separated origin string into a trimmed slice.
// Empty entries and leading/trailing whitespace are discarded.
func parseOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

// isAllowed reports whether origin exactly matches one of the allowed origins.
// It returns false if origin is empty or the allowlist is nil.
func isAllowed(origin string, allowed []string) bool {
	if origin == "" || len(allowed) == 0 {
		return false
	}
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}
