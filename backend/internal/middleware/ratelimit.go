package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// visitor tracks request count and window expiry for a single IP address.
type visitor struct {
	count   int
	resetAt time.Time
}

// RateLimiter enforces a per-IP request limit using a fixed-window algorithm.
// Each IP gets `limit` requests per `window` (default: 1 minute). When the
// window expires, the counter resets. A background goroutine periodically
// evicts expired entries to prevent unbounded memory growth.
type RateLimiter struct {
	mu             sync.Mutex
	visitors       map[string]*visitor
	limit          int
	window         time.Duration
	trustedProxies []*net.IPNet
	stop           chan struct{}
}

// NewRateLimiter creates a rate limiter allowing perMinute requests per IP.
// trustedProxies is a comma-separated list of CIDRs whose X-Forwarded-For
// headers will be trusted for extracting the real client IP.
func NewRateLimiter(perMinute int, trustedProxies string) *RateLimiter {
	rl := &RateLimiter{
		visitors:       make(map[string]*visitor),
		limit:          perMinute,
		window:         time.Minute,
		trustedProxies: parseCIDRs(trustedProxies),
		stop:           make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Stop terminates the background cleanup goroutine. Call on shutdown.
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

// Handler returns middleware that rejects requests exceeding the rate limit
// with HTTP 429 Too Many Requests.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.clientIP(r)

		if !rl.allow(ip) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allow checks whether the given IP has remaining capacity in the current window.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]

	if !exists || now.After(v.resetAt) {
		rl.visitors[ip] = &visitor{count: 1, resetAt: now.Add(rl.window)}
		return true
	}

	v.count++
	return v.count <= rl.limit
}

// cleanup periodically evicts expired visitor entries to prevent unbounded
// memory growth from accumulated IP records. It runs every minute and removes
// any visitor whose window has elapsed. The goroutine exits when the stop
// channel is closed via Stop.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, v := range rl.visitors {
				if now.After(v.resetAt) {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// clientIP determines the real client IP by walking the X-Forwarded-For chain
// from right to left, skipping any addresses that belong to trusted proxy
// CIDRs. The rightmost untrusted IP is treated as the true client address.
// If no trusted proxies are configured or the direct connection is not from a
// trusted proxy, RemoteAddr is returned as-is.
func (rl *RateLimiter) clientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	remoteIP := net.ParseIP(remoteHost)
	if remoteIP == nil {
		return r.RemoteAddr
	}

	if !rl.isTrustedProxy(remoteIP) {
		return remoteHost
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		// Walk backwards to find the first IP not from a trusted proxy
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			parsed := net.ParseIP(ip)
			if parsed == nil {
				continue
			}
			if !rl.isTrustedProxy(parsed) {
				return ip
			}
		}
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		if parsed := net.ParseIP(strings.TrimSpace(realIP)); parsed != nil {
			return realIP
		}
	}

	return remoteHost
}

// isTrustedProxy reports whether ip falls within any of the configured
// trusted proxy CIDR ranges.
func (rl *RateLimiter) isTrustedProxy(ip net.IP) bool {
	for _, cidr := range rl.trustedProxies {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// parseCIDRs parses a comma-separated list of CIDR notations into IPNet values.
// Bare IP addresses without a prefix length are automatically converted to
// /32 (IPv4) or /128 (IPv6). Malformed entries are silently skipped.
func parseCIDRs(raw string) []*net.IPNet {
	if raw == "" {
		return nil
	}
	var nets []*net.IPNet
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// Support bare IPs by appending /32 or /128
		if !strings.Contains(s, "/") {
			if strings.Contains(s, ":") {
				s += "/128"
			} else {
				s += "/32"
			}
		}
		_, cidr, err := net.ParseCIDR(s)
		if err != nil {
			continue
		}
		nets = append(nets, cidr)
	}
	return nets
}
