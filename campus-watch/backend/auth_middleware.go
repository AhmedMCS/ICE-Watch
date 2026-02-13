package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type contextKey string

const userIDKey contextKey = "userID"

// GetUserIDFromContext extracts the user ID from the request context
func GetUserIDFromContext(r *http.Request) string {
	if uid, ok := r.Context().Value(userIDKey).(string); ok {
		return uid
	}
	return ""
}

// OptionalAuthMiddleware extracts and validates auth tokens but does not reject unauthenticated requests.
// Used in "optional" and "grace" auth phases.
func (h *Handlers) OptionalAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			next(w, r)
			return
		}

		claims, err := ValidateSessionToken(token, h.authConfig.JWTSecret)
		if err != nil {
			// Token invalid but auth is optional — proceed without user context
			next(w, r)
			return
		}

		// Verify session exists in Redis
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		session, err := h.redis.GetSession(ctx, claims.ID)
		if err != nil || session == nil {
			next(w, r)
			return
		}

		// Set user ID in context
		ctxWithUser := context.WithValue(r.Context(), userIDKey, claims.UserID)
		next(w, r.WithContext(ctxWithUser))
	}
}

// RequireAuthMiddleware validates auth tokens and rejects unauthenticated requests with 401.
// Used in "enforced" auth phase.
func (h *Handlers) RequireAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		claims, err := ValidateSessionToken(token, h.authConfig.JWTSecret)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Verify session exists in Redis
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		session, err := h.redis.GetSession(ctx, claims.ID)
		if err != nil || session == nil {
			http.Error(w, "Session expired or invalid", http.StatusUnauthorized)
			return
		}

		// Set user ID in context
		ctxWithUser := context.WithValue(r.Context(), userIDKey, claims.UserID)
		next(w, r.WithContext(ctxWithUser))
	}
}

// UserRateLimitMiddleware checks user-based rate limits (5 reports/24h) if authenticated,
// otherwise falls back to the existing IP-based rate limiter.
func (h *Handlers) UserRateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserIDFromContext(r)

		if userID != "" {
			// Authenticated user: check user-based rate limit
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			allowed, err := h.redis.CheckUserRateLimit(ctx, userID)
			if err != nil {
				log.Printf("Error checking user rate limit: %v", err)
				// On error, allow the request through
				next(w, r)
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", "3600")
				http.Error(w, "Rate limit exceeded. Maximum 5 reports per 24 hours.", http.StatusTooManyRequests)
				return
			}
		} else {
			// Unauthenticated: fall back to IP-based rate limiting via Redis
			ip := getClientIP(r)
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			allowed, err := h.redis.CheckIPRateLimit(ctx, ip, h.rateLimitMinutes)
			if err != nil {
				log.Printf("Error checking IP rate limit: %v", err)
				// Fail-open on Redis error
				next(w, r)
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(h.rateLimitMinutes*60))
				http.Error(w, "Rate limit exceeded. Please wait before submitting another report.", http.StatusTooManyRequests)
				return
			}
		}

		next(w, r)
	}
}

// getVoterIdentity returns a stable identity for vote deduplication.
// Prefers authenticated userID, falls back to IP.
func getVoterIdentity(r *http.Request) string {
	if uid := GetUserIDFromContext(r); uid != "" {
		return "user:" + uid
	}
	return "ip:" + getClientIP(r)
}

// ActionRateLimitMiddleware rate-limits an action endpoint per identity.
func (h *Handlers) ActionRateLimitMiddleware(action string, maxCount int, windowMinutes int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity := getVoterIdentity(r)

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		allowed, retryAfter, err := h.redis.CheckActionRateLimit(ctx, identity, action, maxCount, windowMinutes)
		if err != nil {
			log.Printf("Error checking action rate limit: %v", err)
			next(w, r)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			http.Error(w, "Rate limit exceeded. Please wait before performing this action again.", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// extractBearerToken extracts the token from the Authorization header
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
