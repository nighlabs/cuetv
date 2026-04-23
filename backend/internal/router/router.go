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
	queueHandler := handlers.NewQueueHandler(queueService)
	configHandler := handlers.NewConfigHandler(configService)
	eventsHandler := handlers.NewEventsHandler(sseBroker, queries)
	// Parse allowed origins for WebSocket origin validation
	var wsOrigins []string
	if cfg.AllowedOrigins != "" {
		for _, o := range strings.Split(cfg.AllowedOrigins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				wsOrigins = append(wsOrigins, o)
			}
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

			// Authenticated endpoints
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(authService))

				r.Get("/", sessionHandler.Get)

				r.Get("/queue", queueHandler.List)
				r.Post("/queue", queueHandler.Add)
				r.Patch("/queue", queueHandler.Reorder)
				r.Delete("/queue/{itemId}", queueHandler.Delete)

				r.Post("/playback", playbackHandler.Command)

				r.Get("/config", configHandler.Get)
				r.Patch("/config", configHandler.Update)
			})
		})
	})

	return r
}
