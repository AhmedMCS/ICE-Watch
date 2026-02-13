package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return fallback
}

func main() {
	// Load configuration from environment
	port := getEnv("PORT", "8080")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	alertRadius := getEnvFloat("ALERT_RADIUS_MILES", 1.0)
	rateLimitMinutes := getEnvInt("RATE_LIMIT_MINUTES", 5)
	sightingTTLHours := getEnvInt("SIGHTING_TTL_HOURS", 24)

	log.Printf("Starting Campus Watch server...")
	log.Printf("Configuration: port=%s, alertRadius=%.1f miles, rateLimit=%d min, ttl=%d hours",
		port, alertRadius, rateLimitMinutes, sightingTTLHours)

	// Initialize Redis client
	redisClient, err := NewRedisClient(redisURL, sightingTTLHours)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Printf("Connected to Redis")

	// Initialize components
	hub := NewHub(alertRadius)
	authConfig := NewAuthConfig()
	handlers := NewHandlers(hub, redisClient, authConfig, rateLimitMinutes)
	log.Printf("Auth phase: %s", authConfig.AuthPhase)

	// Start the hub
	go hub.Run()

	// Start periodic cleanup
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			redisClient.CleanupExpired(ctx)
			cancel()
		}
	}()

	// Build the sighting POST handler chain based on auth phase
	var sightingPostHandler http.HandlerFunc
	switch authConfig.AuthPhase {
	case "enforced":
		sightingPostHandler = handlers.RequireAuthMiddleware(
			handlers.UserRateLimitMiddleware(handlers.CreateSightingHandler),
		)
	default: // "optional" or "grace"
		sightingPostHandler = handlers.OptionalAuthMiddleware(
			handlers.UserRateLimitMiddleware(handlers.CreateSightingHandler),
		)
	}

	// Register routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", CORSMiddleware(handlers.HealthHandler))
	// Wrap action endpoints with optional auth (for identity) + action rate limits
	voteHandler := handlers.OptionalAuthMiddleware(
		handlers.ActionRateLimitMiddleware("vote", 20, 5, handlers.VoteHandler),
	)
	confirmHandler := handlers.OptionalAuthMiddleware(
		handlers.ActionRateLimitMiddleware("vote", 20, 5, handlers.ConfirmSightingHandler),
	)
	allClearHandler := handlers.OptionalAuthMiddleware(
		handlers.ActionRateLimitMiddleware("vote", 20, 5, handlers.AllClearHandler),
	)
	requestSupportHandler := handlers.OptionalAuthMiddleware(
		handlers.ActionRateLimitMiddleware("request_support", 3, 10, handlers.RequestSupportHandler),
	)
	respondHandler := handlers.OptionalAuthMiddleware(
		handlers.ActionRateLimitMiddleware("respond", 5, 5, handlers.RespondToSupportHandler),
	)
	deleteHandler := handlers.OptionalAuthMiddleware(handlers.DeleteSightingHandler)

	mux.HandleFunc("/api/sightings", CORSMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetSightingsHandler(w, r)
		case http.MethodPost:
			sightingPostHandler(w, r)
		case http.MethodDelete:
			deleteHandler(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/sightings/vote", CORSMiddleware(voteHandler))
	mux.HandleFunc("/api/sightings/confirm", CORSMiddleware(confirmHandler))
	mux.HandleFunc("/api/sightings/all-clear", CORSMiddleware(allClearHandler))
	mux.HandleFunc("/api/sightings/request-support", CORSMiddleware(requestSupportHandler))
	mux.HandleFunc("/api/sightings/respond", CORSMiddleware(respondHandler))
	mux.HandleFunc("/api/sightings/cancel-support", CORSMiddleware(handlers.CancelSupportHandler))
	mux.HandleFunc("/api/feedback", CORSMiddleware(handlers.FeedbackHandler))
	mux.HandleFunc("/ws", handlers.WebSocketHandler)

	// Auth routes
	mux.HandleFunc("/auth/google/login", CORSMiddleware(handlers.GoogleLoginHandler))
	mux.HandleFunc("/auth/status", CORSMiddleware(handlers.OptionalAuthMiddleware(handlers.AuthStatusHandler)))
	mux.HandleFunc("/auth/verify", CORSMiddleware(handlers.VerifyHandler))
	mux.HandleFunc("/auth/logout", CORSMiddleware(handlers.LogoutHandler))

	// Create server with timeouts
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	log.Printf("Server listening on :%s", port)

	<-done
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}
