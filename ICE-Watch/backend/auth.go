package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	GoogleClientID       string
	JWTSecret            []byte
	AllowedEmailDomain   string
	SessionDurationHours int
	UserTTLDays          int
	AuthPhase            string // optional, grace, enforced
}

// NewAuthConfig creates an AuthConfig from environment variables
func NewAuthConfig() *AuthConfig {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	jwtSecret := os.Getenv("JWT_SECRET")

	if googleClientID == "" {
		log.Println("WARNING: GOOGLE_CLIENT_ID not set - auth endpoints will reject all requests")
	}
	if jwtSecret == "" {
		log.Println("WARNING: JWT_SECRET not set - auth endpoints will reject all requests")
	}

	sessionHours, err := strconv.Atoi(os.Getenv("SESSION_DURATION_HOURS"))
	if err != nil || sessionHours <= 0 {
		sessionHours = 168 // 7 days
	}

	userTTLDays, err := strconv.Atoi(os.Getenv("USER_TTL_DAYS"))
	if err != nil || userTTLDays <= 0 {
		userTTLDays = 30
	}

	authPhase := os.Getenv("AUTH_PHASE")
	if authPhase == "" {
		authPhase = "optional"
	}

	return &AuthConfig{
		GoogleClientID:       googleClientID,
		JWTSecret:            []byte(jwtSecret),
		AllowedEmailDomain:   getEnv("ALLOWED_EMAIL_DOMAIN", "scarletmail.rutgers.edu"),
		SessionDurationHours: sessionHours,
		UserTTLDays:          userTTLDays,
		AuthPhase:            authPhase,
	}
}

// VerifyGoogleIDToken verifies a Google ID token by calling Google's tokeninfo endpoint
func VerifyGoogleIDToken(idToken, expectedClientID string) (*GoogleTokenPayload, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token with Google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google token verification failed (status %d): %s", resp.StatusCode, string(body))
	}

	var payload GoogleTokenPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode Google token response: %w", err)
	}

	if payload.Aud != expectedClientID {
		return nil, fmt.Errorf("token audience mismatch: got %s, expected %s", payload.Aud, expectedClientID)
	}

	return &payload, nil
}

// ValidateEmailDomain checks if the email belongs to the allowed domain (case-insensitive)
func ValidateEmailDomain(email, allowedDomain string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	allowedDomain = strings.ToLower(strings.TrimSpace(allowedDomain))
	return strings.HasSuffix(email, "@"+allowedDomain)
}

// HashEmail returns a hex-encoded SHA-256 hash of the lowercased, trimmed email
func HashEmail(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}

// GenerateSessionToken creates a signed JWT with the user ID and expiration
func GenerateSessionToken(userID string, secret []byte, duration time.Duration) (string, string, time.Time, error) {
	jti := uuid.New().String()
	expiresAt := time.Now().Add(duration)

	claims := SessionClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, jti, expiresAt, nil
}

// ValidateSessionToken parses and validates a JWT, returning the claims
func ValidateSessionToken(tokenString string, secret []byte) (*SessionClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*SessionClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
