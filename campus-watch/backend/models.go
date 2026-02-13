package main

import (
	"time"

	"github.com/gorilla/websocket"
)

// Sighting represents an ICE sighting report
type Sighting struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	ImageData   string    `json:"image_data,omitempty"` // Base64 encoded, max 1MB
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	Confidence  string    `json:"confidence"` // confirmed/likely/uncertain
	Timestamp   time.Time `json:"timestamp"`
	ExpiresAt   time.Time `json:"expires_at"` // 24 hours from creation

	// Category & Details
	Category          string `json:"category"`                    // vehicle, on_foot, other
	VehicleType       string `json:"vehicle_type,omitempty"`      // van, suv, sedan, unmarked, multiple
	ApproxCount       string `json:"approx_count,omitempty"`      // 1-2, 3-5, 5+
	MovementDirection string `json:"movement_direction,omitempty"` // north, south, east, west, stationary

	// Community Validation (legacy, kept for migration)
	ConfirmationCount int       `json:"confirmation_count,omitempty"`
	LastConfirmedAt   time.Time `json:"last_confirmed_at,omitempty"`

	// All Clear System (legacy, kept for migration)
	AllClearCount     int    `json:"all_clear_count,omitempty"`
	AllClearThreshold int    `json:"all_clear_threshold,omitempty"`
	Status            string `json:"status"` // active, support_requested

	// Crowdsourced voting (replaces ConfirmationCount/AllClearCount)
	StillThereCount    int       `json:"still_there_count"`
	NoLongerThereCount int       `json:"no_longer_there_count"`
	LastVoteAt         time.Time `json:"last_vote_at,omitempty"`

	// Verification status (independent from Status field)
	VerificationStatus string `json:"verification_status"` // unverified, community_verified, disputed, removed

	// Computed scores
	ConfidenceScore float64 `json:"confidence_score"` // 0.0-1.0, decay-weighted
	FeedScore       float64 `json:"feed_score"`       // for sorting

	// Support Request System
	SupportRequested   bool      `json:"support_requested"`
	SupportRequestedAt time.Time `json:"support_requested_at,omitempty"`
	RespondersCount    int       `json:"responders_count"`
	RespondersNeeded   int       `json:"responders_needed"` // Default: 3

}

// Client represents a connected WebSocket client
type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Lat  float64
	Lng  float64
	Send chan []byte
}

// LocationUpdate represents a client location update message
type LocationUpdate struct {
	Type string  `json:"type"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

// SightingRequest represents the incoming sighting report request
type SightingRequest struct {
	Description string  `json:"description"`
	ImageData   string  `json:"image_data,omitempty"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Confidence  string  `json:"confidence"`
	Category    string  `json:"category"`
	VehicleType string  `json:"vehicle_type,omitempty"`
	ApproxCount       string  `json:"approx_count,omitempty"`
	MovementDirection string  `json:"movement_direction,omitempty"`
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string   `json:"type"`
	Payload Sighting `json:"payload,omitempty"`
}

// SightingUpdate represents an update to an existing sighting
type SightingUpdate struct {
	SightingID         string    `json:"sighting_id"`
	ConfirmationCount  int       `json:"confirmation_count,omitempty"`
	AllClearCount      int       `json:"all_clear_count,omitempty"`
	StillThereCount    int       `json:"still_there_count"`
	NoLongerThereCount int       `json:"no_longer_there_count"`
	VerificationStatus string    `json:"verification_status"`
	Status             string    `json:"status"`
	LastVoteAt         time.Time `json:"last_vote_at,omitempty"`
	ConfidenceScore    float64   `json:"confidence_score"`
	FeedScore          float64   `json:"feed_score"`
	LastConfirmedAt    time.Time `json:"last_confirmed_at,omitempty"`
}

// VoteRequest represents a vote submission
type VoteRequest struct {
	Type string `json:"type"` // "still_there" or "no_longer_there"
}

// VoteEntry represents a recorded vote for decay calculations
type VoteEntry struct {
	Identity  string
	VoteType  string
	Timestamp time.Time
}

// WebSocketUpdateMessage represents an update message sent over WebSocket
type WebSocketUpdateMessage struct {
	Type    string         `json:"type"`
	Payload SightingUpdate `json:"payload"`
}

// SupportResponse represents a user responding to a support request
type SupportResponse struct {
	SightingID  string    `json:"sighting_id"`
	ResponderID string    `json:"responder_id"` // Anonymous ID for this session
	Timestamp   time.Time `json:"timestamp"`
	ETA         int       `json:"eta"` // Minutes
}

// Feedback represents user feedback submission
type Feedback struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`    // bug, feature, general, privacy
	Message    string    `json:"message"`
	DeviceInfo string    `json:"device_info,omitempty"` // Optional
	Timestamp  time.Time `json:"timestamp"`
	Status     string    `json:"status"` // new, reviewing, resolved
}

// SupportRequestMessage represents a high-priority support request broadcast
type SupportRequestMessage struct {
	Type     string   `json:"type"`
	Sighting Sighting `json:"sighting"`
	Radius   float64  `json:"radius"` // Broadcast radius in miles
}
