package router

import (
	"database/sql"
	"strings"

	"github.com/cuetv/backend/internal/broker"
	"github.com/cuetv/backend/internal/config"
	"github.com/cuetv/backend/internal/db"
	"github.com/cuetv/backend/internal/handlers"
	"github.com/cuetv/backend/internal/middleware"
	"github.com/cuetv/backend/internal/services"
	"github.com/go-chi/chi/v5"
)

// New wires all dependencies (database, brokers, services, handlers) and
// defines the full route table for the application.
func New(cfg *config.Config, database *sql.DB) *chi.Mux {
	queries := db.New(database)

	// Real-time brokers
	sseBroker := broker.NewSSEBroker()
	wsHub := broker.NewWSHub()

	// Services — all business logic lives here
	authService := services.NewAuthService(
		cfg.JWTSecret,
		cfg.AdminPortalPassword,
		cfg.AdminTokenDuration,
		cfg.FriendTokenDuration,
	)
	sessionService := services.NewSessionService(database, queries, authService)
	queueService := services.NewQueueService(database, queries, wsHub)
	configService := services.NewConfigService(database, queries, wsHub)
	playbackService := services.NewPlaybackService(queries, sseBroker, wsHub)

	// Handlers — thin HTTP layer, delegates to services
	authHandler := handlers.NewAuthHandler(authService)
	sessionHandler := handlers.NewSessionHandler(sessionService, authService)
	queueHandler := handlers.NewQueueHandler(queueService, queries, authService)
	configHandler := handlers.NewConfigHandler(configService, queries, authService)
	eventsHandler := handlers.NewEventsHandler(sseBroker, queries)
	// Parse allowed origins for WebSocket origin validation. The nhooyr.io/websocket
	// library's OriginPatterns expects host[:port] patterns (e.g. "localhost:3000"),
	// not full URLs, so we strip the scheme from ALLOWED_ORIGINS.
	var wsOrigins []string
	if cfg.AllowedOrigins != "" {
		for _, o := range strings.Split(cfg.AllowedOrigins, ",") {
			o = strings.TrimSpace(o)
			if o == "" {
				continue
			}
			// Strip scheme (http:// or https://) to get host:port pattern
			o = strings.TrimPrefix(o, "https://")
			o = strings.TrimPrefix(o, "http://")
			wsOrigins = append(wsOrigins, o)
		}
	}
	wsHandler := handlers.NewWebSocketHandler(wsHub, authService, wsOrigins)
	playbackHandler := handlers.NewPlaybackHandler(playbackService, queries)

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitPerMinute, cfg.TrustedProxies)

	// Route definitions
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logging)
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handlers.Health)

		// Auth - rate limited
		r.With(rateLimiter.Handler).Post("/admin/verify", authHandler.Verify)

		// Sessions - public endpoints (rate limited)
		r.With(rateLimiter.Handler).Post("/sessions", sessionHandler.Create)
		r.With(rateLimiter.Handler).Post("/sessions/join", sessionHandler.Join)

		// Sessions - authenticated endpoints
		r.With(middleware.Auth(authService)).Post("/sessions/rejoin", sessionHandler.Rejoin)

		r.Route("/sessions/{id}", func(r chi.Router) {
			// SSE endpoint - viewer token auth via query param
			r.Get("/events", eventsHandler.SSEStream)

			// WebSocket endpoint - JWT auth via query param
			r.Get("/queue-events", wsHandler.HandleQueueEvents)

			// Video ended - viewer token auth via query param
			r.Post("/video-ended", playbackHandler.VideoEnded)

			// Read-only endpoints — accept either JWT or viewer token,
			// so both admin and viewer pages can fetch queue/config.
			r.Get("/queue", queueHandler.List)
			r.Get("/config", configHandler.Get)

			// Write endpoints — require JWT auth
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(authService))

				r.Get("/", sessionHandler.Get)

				r.Post("/queue", queueHandler.Add)
				r.Patch("/queue", queueHandler.Reorder)
				r.Delete("/queue/{itemId}", queueHandler.Delete)

				r.Post("/playback", playbackHandler.Command)

				r.Patch("/config", configHandler.Update)
			})
		})
	})

	return r
}
