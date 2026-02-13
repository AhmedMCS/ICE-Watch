package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// GoogleLoginHandler handles POST /auth/google/login
func (h *Handlers) GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.authConfig.GoogleClientID == "" || len(h.authConfig.JWTSecret) == 0 {
		http.Error(w, "Authentication not configured", http.StatusServiceUnavailable)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.IDToken == "" {
		http.Error(w, "id_token is required", http.StatusBadRequest)
		return
	}

	// Verify the Google ID token
	payload, err := VerifyGoogleIDToken(req.IDToken, h.authConfig.GoogleClientID)
	if err != nil {
		log.Printf("Google token verification failed: %v", err)
		http.Error(w, "Invalid Google token", http.StatusUnauthorized)
		return
	}

	// Validate email domain
	if !ValidateEmailDomain(payload.Email, h.authConfig.AllowedEmailDomain) {
		http.Error(w, "Only @"+h.authConfig.AllowedEmailDomain+" emails are allowed", http.StatusForbidden)
		return
	}

	// Hash the email
	emailHash := HashEmail(payload.Email)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Find or create user
	userTTL := time.Duration(h.authConfig.UserTTLDays) * 24 * time.Hour
	user, err := h.redis.FindOrCreateUser(ctx, emailHash, userTTL)
	if err != nil {
		log.Printf("Error finding/creating user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update user activity
	if err := h.redis.UpdateUserActivity(ctx, user, userTTL); err != nil {
		log.Printf("Error updating user activity: %v", err)
	}

	// Generate session token
	sessionDuration := time.Duration(h.authConfig.SessionDurationHours) * time.Hour
	tokenString, jti, expiresAt, err := GenerateSessionToken(user.ID, h.authConfig.JWTSecret, sessionDuration)
	if err != nil {
		log.Printf("Error generating session token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store session in Redis
	session := &Session{
		Token:      tokenString,
		UserID:     user.ID,
		ExpiresAt:  expiresAt,
		DeviceInfo: req.DeviceInfo,
		CreatedAt:  time.Now(),
	}
	if err := h.redis.StoreSession(ctx, jti, session, sessionDuration); err != nil {
		log.Printf("Error storing session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return auth response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Unix(),
		UserID:    user.ID,
	})
}

// AuthStatusHandler handles GET /auth/status
func (h *Handlers) AuthStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r)
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthStatusResponse{
			Authenticated: false,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	user, err := h.redis.GetUser(ctx, userID)
	if err != nil || user == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthStatusResponse{
			Authenticated: false,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthStatusResponse{
		Authenticated:    true,
		UserID:           user.ID,
		ReportsSubmitted: user.ReportsSubmitted,
	})
}

// VerifyHandler handles POST /auth/verify — validates a token on app startup
func (h *Handlers) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractBearerToken(r)
	if token == "" {
		http.Error(w, "No token provided", http.StatusUnauthorized)
		return
	}

	claims, err := ValidateSessionToken(token, h.authConfig.JWTSecret)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	session, err := h.redis.GetSession(ctx, claims.ID)
	if err != nil || session == nil {
		http.Error(w, "Session expired", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token:     token,
		ExpiresAt: claims.ExpiresAt.Unix(),
		UserID:    claims.UserID,
	})
}

// LogoutHandler handles POST /auth/logout
func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractBearerToken(r)
	if token == "" {
		http.Error(w, "No token provided", http.StatusUnauthorized)
		return
	}

	claims, err := ValidateSessionToken(token, h.authConfig.JWTSecret)
	if err != nil {
		// Even if token is invalid/expired, treat logout as success
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.redis.DeleteSession(ctx, claims.ID); err != nil {
		log.Printf("Error deleting session: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
