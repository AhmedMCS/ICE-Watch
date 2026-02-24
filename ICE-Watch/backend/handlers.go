package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

const maxImageSize = 1024 * 1024 // 1MB

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" || allowedOrigin == "*" {
			return true
		}
		return r.Header.Get("Origin") == allowedOrigin
	},
}

// Handlers holds all HTTP handler dependencies
type Handlers struct {
	hub              *Hub
	redis            *RedisClient
	authConfig       *AuthConfig
	rateLimitMinutes int
}

// NewHandlers creates a new Handlers instance
func NewHandlers(hub *Hub, redis *RedisClient, authConfig *AuthConfig, rateLimitMinutes int) *Handlers {
	return &Handlers{
		hub:              hub,
		redis:            redis,
		authConfig:       authConfig,
		rateLimitMinutes: rateLimitMinutes,
	}
}

// HealthHandler returns the health status
func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.redis.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "redis": "down"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// CreateSightingHandler handles new sighting reports
func (h *Handlers) CreateSightingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to 2MB
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)

	var req SightingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize description: strip HTML tags, limit length
	req.Description = htmlTagRegex.ReplaceAllString(req.Description, "")
	req.Description = strings.TrimSpace(req.Description)
	if len(req.Description) > 500 {
		req.Description = req.Description[:500]
	}

	// Validate required fields
	if req.Lat == 0 && req.Lng == 0 {
		http.Error(w, "Location (lat/lng) is required", http.StatusBadRequest)
		return
	}

	// Validate confidence level
	if req.Confidence == "" {
		req.Confidence = "uncertain"
	}
	if req.Confidence != "confirmed" && req.Confidence != "likely" && req.Confidence != "uncertain" {
		http.Error(w, "Confidence must be: confirmed, likely, or uncertain", http.StatusBadRequest)
		return
	}

	// Validate image size
	if len(req.ImageData) > maxImageSize {
		http.Error(w, "Image too large. Maximum size is 1MB.", http.StatusBadRequest)
		return
	}

	// Validate category (required)
	validCategories := map[string]bool{
		"agents_present": true, "raid_in_progress": true, "questioning": true,
	}
	if !validCategories[req.Category] {
		http.Error(w, "Invalid or missing category", http.StatusBadRequest)
		return
	}

	// Validate description (required, min 10 chars)
	if len(req.Description) < 10 {
		http.Error(w, "Description is required (minimum 10 characters)", http.StatusBadRequest)
		return
	}

	// Validate vehicle_type
	validVehicleTypes := map[string]bool{
		"van": true, "suv": true, "sedan": true, "unmarked": true, "multiple": true, "": true,
	}
	if !validVehicleTypes[req.VehicleType] {
		http.Error(w, "Invalid vehicle type", http.StatusBadRequest)
		return
	}

	// Validate approx_count
	validApproxCounts := map[string]bool{
		"1-2": true, "3-5": true, "5+": true, "": true,
	}
	if !validApproxCounts[req.ApproxCount] {
		http.Error(w, "Invalid approximate count", http.StatusBadRequest)
		return
	}

	// Validate movement_direction
	validDirections := map[string]bool{
		"north": true, "south": true, "east": true, "west": true, "stationary": true, "": true,
	}
	if !validDirections[req.MovementDirection] {
		http.Error(w, "Invalid movement direction", http.StatusBadRequest)
		return
	}

	// Create sighting
	now := time.Now()
	sighting := &Sighting{
		ID:                 uuid.New().String(),
		Description:        req.Description,
		ImageData:          req.ImageData,
		Lat:                req.Lat,
		Lng:                req.Lng,
		Confidence:         req.Confidence,
		Timestamp:          now,
		ExpiresAt:          now.Add(24 * time.Hour),
		Category:           req.Category,
		VehicleType:        req.VehicleType,
		ApproxCount:        req.ApproxCount,
		MovementDirection:  req.MovementDirection,
		Status:             "active",
		AllClearThreshold:  5,
		RespondersNeeded:   3,
		VerificationStatus: "unverified",
		ConfidenceScore:    0.5,
	}
	sighting.FeedScore = computeFeedScore(sighting)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Determine reporter identity key prefix
	userID := GetUserIDFromContext(r)
	var identityKey string
	if userID != "" {
		identityKey = fmt.Sprintf("user:%s", userID)
	} else {
		identityKey = fmt.Sprintf("ip:%s", getClientIP(r))
	}

	// Location deduplication: reject if reporter already submitted near this spot today
	locationKey := fmt.Sprintf("reporter_locations:%s", identityKey)
	nearby, err := h.redis.CheckReporterNearbyLocation(ctx, locationKey, req.Lat, req.Lng, 0.1)
	if err != nil {
		log.Printf("Error checking reporter locations: %v", err)
	} else if nearby {
		http.Error(w, "You've already reported a sighting near this location today.", http.StatusConflict)
		return
	}

	// Atomically check-and-record rate limit BEFORE storage to close the race window
	rateLimitKey := fmt.Sprintf("rate_limit:%s", identityKey)
	allowed, err := h.redis.AtomicCheckAndRecordRateLimit(ctx, rateLimitKey, 3, 24*60)
	if err != nil {
		log.Printf("Error in atomic rate limit: %v", err)
	} else if !allowed {
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "Rate limit exceeded. Maximum 3 reports per 24 hours.", http.StatusTooManyRequests)
		return
	}

	// Store in Redis
	if err := h.redis.StoreSighting(ctx, sighting); err != nil {
		log.Printf("Error storing sighting: %v", err)
		http.Error(w, "Failed to store sighting", http.StatusInternalServerError)
		return
	}

	// Record reporter location for deduplication (non-critical, best-effort)
	if err := h.redis.RecordReporterLocation(ctx, locationKey, req.Lat, req.Lng, sighting.ID); err != nil {
		log.Printf("Error recording reporter location: %v", err)
	}

	// Update user report count (non-critical, best-effort)
	if userID != "" {
		userTTL := time.Duration(h.authConfig.UserTTLDays) * 24 * time.Hour
		if err := h.redis.IncrementUserReports(ctx, userID, userTTL); err != nil {
			log.Printf("Error incrementing user reports: %v", err)
		}
	}

	// Store creator identity for withdrawal
	creatorIdentity := getVoterIdentity(r)
	if err := h.redis.StoreSightingCreator(ctx, sighting.ID, creatorIdentity); err != nil {
		log.Printf("Error storing sighting creator: %v", err)
	}

	// Broadcast to nearby clients
	h.hub.BroadcastSighting(sighting)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sighting)
}

// GetSightingsHandler returns active sightings with optional geo-filtering and pagination
func (h *Handlers) GetSightingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	query := r.URL.Query()

	// Check for geo-filtering params
	latStr := query.Get("lat")
	lngStr := query.Get("lng")
	radiusStr := query.Get("radius")

	if latStr != "" && lngStr != "" && radiusStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			http.Error(w, "Invalid lat parameter", http.StatusBadRequest)
			return
		}
		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			http.Error(w, "Invalid lng parameter", http.StatusBadRequest)
			return
		}
		radius, err := strconv.ParseFloat(radiusStr, 64)
		if err != nil || radius <= 0 {
			http.Error(w, "Invalid radius parameter", http.StatusBadRequest)
			return
		}

		sightings, err := h.redis.GetNearbySightings(ctx, lat, lng, radius)
		if err != nil {
			log.Printf("Error getting nearby sightings: %v", err)
			http.Error(w, "Failed to retrieve sightings", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sightings)
		return
	}

	// Check for pagination params
	limitStr := query.Get("limit")
	beforeStr := query.Get("before")

	limit := 0
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
	}

	before := float64(time.Now().Unix() + 1) // +1 so exclusive bound still catches "now"
	if beforeStr != "" {
		var err error
		before, err = strconv.ParseFloat(beforeStr, 64)
		if err != nil {
			http.Error(w, "Invalid before parameter", http.StatusBadRequest)
			return
		}
	}

	sightings, hasMore, err := h.redis.GetActiveSightings(ctx, limit, before)
	if err != nil {
		log.Printf("Error getting sightings: %v", err)
		http.Error(w, "Failed to retrieve sightings", http.StatusInternalServerError)
		return
	}

	// If pagination is active, return paginated response
	if limit > 0 {
		var nextCursor string
		if hasMore && len(sightings) > 0 {
			nextCursor = strconv.FormatInt(sightings[len(sightings)-1].Timestamp.Unix(), 10)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sightings":   sightings,
			"next_cursor": nextCursor,
			"has_more":    hasMore,
		})
		return
	}

	// No pagination — return plain array for backward compatibility
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sightings)
}

// ConfirmSightingHandler wraps VoteHandler for backward compatibility.
// Maps old "confirm" to new "still_there" vote.
func (h *Handlers) ConfirmSightingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Inject vote type into the request for VoteHandler
	r.URL.RawQuery = r.URL.RawQuery + "&_vote_type=still_there"
	h.VoteHandler(w, r)
}

// AllClearHandler wraps VoteHandler for backward compatibility.
// Maps old "all-clear" to new "no_longer_there" vote.
func (h *Handlers) AllClearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.URL.RawQuery = r.URL.RawQuery + "&_vote_type=no_longer_there"
	h.VoteHandler(w, r)
}

// VoteHandler handles the unified voting endpoint.
// POST /api/sightings/vote?id={id} with body {"type": "still_there"|"no_longer_there"}
// Also called by legacy ConfirmSightingHandler/AllClearHandler via _vote_type query param.
func (h *Handlers) VoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sightingID := r.URL.Query().Get("id")
	if sightingID == "" {
		http.Error(w, "Sighting ID is required", http.StatusBadRequest)
		return
	}

	// Determine vote type: from query param (legacy wrapper) or request body
	voteType := r.URL.Query().Get("_vote_type")
	if voteType == "" {
		var voteReq VoteRequest
		if err := json.NewDecoder(r.Body).Decode(&voteReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		voteType = voteReq.Type
	}

	if voteType != "still_there" && voteType != "no_longer_there" {
		http.Error(w, `Vote type must be "still_there" or "no_longer_there"`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	identity := getVoterIdentity(r)

	// Check existing vote
	existingVote, err := h.redis.GetUserVote(ctx, sightingID, identity)
	if err != nil {
		log.Printf("Error checking existing vote: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Same vote = 409 Conflict
	if existingVote == voteType {
		http.Error(w, "You have already cast this vote", http.StatusConflict)
		return
	}

	// Fetch sighting
	sighting, err := h.redis.GetSighting(ctx, sightingID)
	if err != nil {
		log.Printf("Error getting sighting: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sighting == nil {
		http.Error(w, "Sighting not found", http.StatusNotFound)
		return
	}

	// Handle vote change: decrement old count
	if existingVote != "" {
		if existingVote == "still_there" {
			sighting.StillThereCount--
			if sighting.StillThereCount < 0 {
				sighting.StillThereCount = 0
			}
		} else {
			sighting.NoLongerThereCount--
			if sighting.NoLongerThereCount < 0 {
				sighting.NoLongerThereCount = 0
			}
		}
		// Remove old vote from history
		h.redis.RemoveVoteHistory(ctx, sightingID, identity, existingVote)
	}

	// Increment new vote count
	if voteType == "still_there" {
		sighting.StillThereCount++
		// Also update legacy fields for backward compat
		sighting.ConfirmationCount = sighting.StillThereCount
		sighting.LastConfirmedAt = time.Now()
	} else {
		sighting.NoLongerThereCount++
		sighting.AllClearCount = sighting.NoLongerThereCount
	}

	now := time.Now()
	sighting.LastVoteAt = now

	// Record vote
	if err := h.redis.RecordVote(ctx, sightingID, identity, voteType); err != nil {
		log.Printf("Error recording vote: %v", err)
		http.Error(w, "Failed to record vote", http.StatusInternalServerError)
		return
	}

	// Record vote history for decay calculations
	if err := h.redis.RecordVoteHistory(ctx, sightingID, identity, voteType); err != nil {
		log.Printf("Error recording vote history: %v", err)
	}

	// Record action for rate limiting
	h.redis.RecordAction(ctx, identity, "vote", 5)

	// Recalculate confidence score from vote history with decay
	entries, err := h.redis.GetVoteHistory(ctx, sightingID)
	if err != nil {
		log.Printf("Error getting vote history: %v", err)
	} else {
		sighting.ConfidenceScore = computeConfidenceScore(entries)
	}

	// Evaluate verification status transitions
	evaluateVerificationStatus(sighting)

	// Compute feed score
	sighting.FeedScore = computeFeedScore(sighting)

	// Update sighting in Redis with appropriate TTL
	switch sighting.VerificationStatus {
	case "removed":
		if err := h.redis.UpdateSightingWithTTL(ctx, sighting, 30*time.Minute); err != nil {
			log.Printf("Error updating sighting with TTL: %v", err)
			http.Error(w, "Failed to update sighting", http.StatusInternalServerError)
			return
		}
	case "community_verified":
		// Extend TTL to at least 24h from now
		if err := h.redis.UpdateSightingWithTTL(ctx, sighting, 24*time.Hour); err != nil {
			log.Printf("Error updating sighting with TTL: %v", err)
			http.Error(w, "Failed to update sighting", http.StatusInternalServerError)
			return
		}
	default:
		if err := h.redis.UpdateSighting(ctx, sighting); err != nil {
			log.Printf("Error updating sighting: %v", err)
			http.Error(w, "Failed to update sighting", http.StatusInternalServerError)
			return
		}
	}

	// Broadcast update
	update := &SightingUpdate{
		SightingID:         sighting.ID,
		StillThereCount:    sighting.StillThereCount,
		NoLongerThereCount: sighting.NoLongerThereCount,
		VerificationStatus: sighting.VerificationStatus,
		Status:             sighting.Status,
		LastVoteAt:         sighting.LastVoteAt,
		ConfidenceScore:    sighting.ConfidenceScore,
		FeedScore:          sighting.FeedScore,
		// Legacy fields
		ConfirmationCount: sighting.ConfirmationCount,
		AllClearCount:     sighting.AllClearCount,
		LastConfirmedAt:   sighting.LastConfirmedAt,
	}
	h.hub.BroadcastSightingUpdate(update)

	// Return updated sighting with user's current vote
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sighting":  sighting,
		"your_vote": voteType,
	})
}

// computeConfidenceScore calculates a decay-weighted confidence score from vote history.
// weight(vote) = exp(-0.1 * hours_since_vote)
// confidenceScore = weightedStillThere / (weightedStillThere + weightedNoLongerThere)
func computeConfidenceScore(entries []VoteEntry) float64 {
	if len(entries) == 0 {
		return 0.5 // neutral when no votes
	}

	var weightedStillThere, weightedNoLongerThere float64
	now := time.Now()

	for _, e := range entries {
		hours := now.Sub(e.Timestamp).Hours()
		weight := math.Exp(-0.1 * hours)

		if e.VoteType == "still_there" {
			weightedStillThere += weight
		} else {
			weightedNoLongerThere += weight
		}
	}

	total := weightedStillThere + weightedNoLongerThere
	if total == 0 {
		return 0.5
	}
	return weightedStillThere / total
}

// evaluateVerificationStatus transitions the verification status based on current state.
func evaluateVerificationStatus(s *Sighting) {
	score := s.ConfidenceScore

	// Compute effective counts from decay (approximate via score * total)
	totalCount := s.StillThereCount + s.NoLongerThereCount
	effectiveStillThere := score * float64(totalCount)
	effectiveNoLongerThere := (1 - score) * float64(totalCount)

	switch s.VerificationStatus {
	case "unverified":
		if effectiveStillThere >= 3 && score >= 0.7 {
			s.VerificationStatus = "community_verified"
		}
	case "community_verified":
		if score < 0.7 {
			s.VerificationStatus = "disputed"
		}
	case "disputed":
		if score >= 0.7 {
			s.VerificationStatus = "community_verified"
		}
		if effectiveNoLongerThere >= 3 && score < 0.3 {
			s.VerificationStatus = "removed"
		}
	case "removed":
		// Allow resurrection on new still_there vote
		if score >= 0.3 {
			s.VerificationStatus = "disputed"
		}
	}

	// Any state -> removed
	if s.VerificationStatus != "removed" && effectiveNoLongerThere >= 3 && score < 0.3 {
		s.VerificationStatus = "removed"
	}
}

// computeFeedScore calculates the feed sorting score for a sighting.
// feedScore = (statusWeight * 40) + (confidenceScore * 20) + (voteRecency * 25) + (timeDecay * 10)
func computeFeedScore(s *Sighting) float64 {
	// Status weight
	var statusWeight float64
	switch {
	case s.Status == "support_requested":
		statusWeight = 1.0
	case s.VerificationStatus == "community_verified":
		statusWeight = 0.8
	case s.VerificationStatus == "unverified":
		statusWeight = 0.6
	case s.VerificationStatus == "disputed":
		statusWeight = 0.3
	case s.VerificationStatus == "removed":
		statusWeight = 0.1
	default:
		statusWeight = 0.6
	}

	// Vote recency
	var voteRecency float64
	if !s.LastVoteAt.IsZero() {
		hoursSinceVote := time.Since(s.LastVoteAt).Hours()
		voteRecency = math.Exp(-0.2 * hoursSinceVote)
	}

	// Time decay
	hoursSinceCreation := time.Since(s.Timestamp).Hours()
	timeDecay := math.Exp(-0.05 * hoursSinceCreation)

	return (statusWeight * 40) + (s.ConfidenceScore * 20) + (voteRecency * 25) + (timeDecay * 10)
}

// WebSocketHandler handles WebSocket connections
func (h *Handlers) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		Hub:  h.hub,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	h.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// CORSMiddleware adds CORS headers to responses
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "*" // Default for development
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// RequestSupportHandler handles support requests for a sighting
func (h *Handlers) RequestSupportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sightingID := r.URL.Query().Get("id")
	if sightingID == "" {
		http.Error(w, "Sighting ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sighting, err := h.redis.GetSighting(ctx, sightingID)
	if err != nil {
		log.Printf("Error getting sighting: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sighting == nil {
		http.Error(w, "Sighting not found", http.StatusNotFound)
		return
	}

	// Update sighting for support request
	sighting.SupportRequested = true
	sighting.SupportRequestedAt = time.Now()
	sighting.Status = "support_requested"
	if sighting.RespondersNeeded == 0 {
		sighting.RespondersNeeded = 3
	}

	if err := h.redis.UpdateSighting(ctx, sighting); err != nil {
		log.Printf("Error updating sighting: %v", err)
		http.Error(w, "Failed to update sighting", http.StatusInternalServerError)
		return
	}

	// Broadcast support request to nearby clients (2 mile radius)
	h.hub.BroadcastSupportRequest(sighting, 2.0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"sighting": sighting,
	})
}

// RespondToSupportHandler handles responses to support requests.
// Deduplicates by server-derived identity (userID or IP) instead of client-provided responderID.
func (h *Handlers) RespondToSupportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sightingID := r.URL.Query().Get("id")
	if sightingID == "" {
		http.Error(w, "Sighting ID is required", http.StatusBadRequest)
		return
	}

	var response SupportResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate ETA range
	if response.ETA < 1 || response.ETA > 60 {
		response.ETA = 5
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Use server-derived identity for dedup instead of client-provided responderID
	identity := getVoterIdentity(r)
	response.ResponderID = identity

	hasResponded, err := h.redis.HasResponded(ctx, sightingID, identity)
	if err != nil {
		log.Printf("Error checking responder: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if hasResponded {
		http.Error(w, "Already responded to this support request", http.StatusConflict)
		return
	}

	sighting, err := h.redis.GetSighting(ctx, sightingID)
	if err != nil {
		log.Printf("Error getting sighting: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sighting == nil {
		http.Error(w, "Sighting not found", http.StatusNotFound)
		return
	}

	sighting.RespondersCount++

	response.SightingID = sightingID
	response.Timestamp = time.Now()
	if err := h.redis.RecordResponder(ctx, sightingID, identity, &response); err != nil {
		log.Printf("Error recording responder: %v", err)
	}

	if sighting.RespondersCount >= sighting.RespondersNeeded {
		sighting.Status = "active"
		sighting.SupportRequested = false
	}

	if err := h.redis.UpdateSighting(ctx, sighting); err != nil {
		log.Printf("Error updating sighting: %v", err)
		http.Error(w, "Failed to update sighting", http.StatusInternalServerError)
		return
	}

	update := &SightingUpdate{
		SightingID:         sighting.ID,
		StillThereCount:    sighting.StillThereCount,
		NoLongerThereCount: sighting.NoLongerThereCount,
		VerificationStatus: sighting.VerificationStatus,
		Status:             sighting.Status,
		LastVoteAt:         sighting.LastVoteAt,
		ConfidenceScore:    sighting.ConfidenceScore,
		FeedScore:          sighting.FeedScore,
		ConfirmationCount:  sighting.ConfirmationCount,
		AllClearCount:      sighting.AllClearCount,
	}
	h.hub.BroadcastSightingUpdate(update)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"responders_count": sighting.RespondersCount,
	})
}

// CancelSupportHandler cancels a support request
func (h *Handlers) CancelSupportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sightingID := r.URL.Query().Get("id")
	if sightingID == "" {
		http.Error(w, "Sighting ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sighting, err := h.redis.GetSighting(ctx, sightingID)
	if err != nil {
		log.Printf("Error getting sighting: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sighting == nil {
		http.Error(w, "Sighting not found", http.StatusNotFound)
		return
	}

	// Cancel support request
	sighting.SupportRequested = false
	sighting.Status = "active"

	if err := h.redis.UpdateSighting(ctx, sighting); err != nil {
		log.Printf("Error updating sighting: %v", err)
		http.Error(w, "Failed to update sighting", http.StatusInternalServerError)
		return
	}

	// Broadcast update
	update := &SightingUpdate{
		SightingID:         sighting.ID,
		StillThereCount:    sighting.StillThereCount,
		NoLongerThereCount: sighting.NoLongerThereCount,
		VerificationStatus: sighting.VerificationStatus,
		Status:             sighting.Status,
		LastVoteAt:         sighting.LastVoteAt,
		ConfidenceScore:    sighting.ConfidenceScore,
		FeedScore:          sighting.FeedScore,
		ConfirmationCount:  sighting.ConfirmationCount,
		AllClearCount:      sighting.AllClearCount,
	}
	h.hub.BroadcastSightingUpdate(update)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// FeedbackHandler handles feedback submissions
func (h *Handlers) FeedbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)

	var feedback Feedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if feedback.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Validate feedback type
	validTypes := map[string]bool{"bug": true, "feature": true, "general": true, "privacy": true, "": true}
	if !validTypes[feedback.Type] {
		http.Error(w, "Invalid feedback type", http.StatusBadRequest)
		return
	}
	if feedback.Type == "" {
		feedback.Type = "general"
	}

	feedback.ID = uuid.New().String()
	feedback.Timestamp = time.Now()
	feedback.Status = "new"

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.redis.StoreFeedback(ctx, &feedback); err != nil {
		log.Printf("Error storing feedback: %v", err)
		http.Error(w, "Failed to store feedback", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Thank you for your feedback!",
	})
}

// DeleteSightingHandler allows the original creator to withdraw a report within 5 minutes.
func (h *Handlers) DeleteSightingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sightingID := r.URL.Query().Get("id")
	if sightingID == "" {
		http.Error(w, "Sighting ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Verify the requester is the original creator
	identity := getVoterIdentity(r)
	creator, err := h.redis.GetSightingCreator(ctx, sightingID)
	if err != nil {
		log.Printf("Error getting sighting creator: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if creator == "" || creator != identity {
		http.Error(w, "Only the original reporter can withdraw a sighting", http.StatusForbidden)
		return
	}

	// Check grace period (5 minutes)
	sighting, err := h.redis.GetSighting(ctx, sightingID)
	if err != nil {
		log.Printf("Error getting sighting: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sighting == nil {
		http.Error(w, "Sighting not found", http.StatusNotFound)
		return
	}

	if time.Since(sighting.Timestamp) > 5*time.Minute {
		http.Error(w, "Withdrawal period has expired (5 minutes)", http.StatusGone)
		return
	}

	// Delete the sighting
	if err := h.redis.DeleteSighting(ctx, sightingID); err != nil {
		log.Printf("Error deleting sighting: %v", err)
		http.Error(w, "Failed to delete sighting", http.StatusInternalServerError)
		return
	}

	// Broadcast removal to clients
	update := &SightingUpdate{
		SightingID: sightingID,
		Status:     "withdrawn",
	}
	h.hub.BroadcastSightingUpdate(update)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Sighting withdrawn successfully",
	})
}
