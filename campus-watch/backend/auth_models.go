package main

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents an authenticated user (email is never stored in plaintext)
type User struct {
	ID               string    `json:"id"`
	EmailHash        string    `json:"email_hash"`
	EmailVerified    bool      `json:"email_verified"`
	CreatedAt        time.Time `json:"created_at"`
	LastActive       time.Time `json:"last_active"`
	ReportsSubmitted int       `json:"reports_submitted"`
}

// Session represents an active user session
type Session struct {
	Token      string    `json:"token"`
	UserID     string    `json:"user_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	DeviceInfo string    `json:"device_info,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// GoogleTokenPayload represents the decoded Google ID token claims
type GoogleTokenPayload struct {
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Sub           string `json:"sub"`
	HD            string `json:"hd"`
	Aud           string `json:"aud"`
}

// AuthRequest represents the login request from the client
type AuthRequest struct {
	IDToken    string `json:"id_token"`
	DeviceInfo string `json:"device_info,omitempty"`
}

// AuthResponse represents the login response sent to the client
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	UserID    string `json:"user_id"`
}

// AuthStatusResponse represents the auth status check response
type AuthStatusResponse struct {
	Authenticated    bool   `json:"authenticated"`
	UserID           string `json:"user_id,omitempty"`
	ReportsSubmitted int    `json:"reports_submitted,omitempty"`
	ExpiresAt        int64  `json:"expires_at,omitempty"`
}

// SessionClaims extends JWT registered claims with a user ID
type SessionClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}
