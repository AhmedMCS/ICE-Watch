package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis client with application-specific methods
type RedisClient struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisClient creates a new Redis client wrapper
func NewRedisClient(redisURL string, ttlHours int) (*RedisClient, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: client,
		ttl:    time.Duration(ttlHours) * time.Hour,
	}, nil
}

// StoreSighting saves a sighting to Redis with TTL
func (rc *RedisClient) StoreSighting(ctx context.Context, sighting *Sighting) error {
	key := fmt.Sprintf("sighting:%s", sighting.ID)

	data, err := json.Marshal(sighting)
	if err != nil {
		return fmt.Errorf("failed to marshal sighting: %w", err)
	}

	// Store the sighting with TTL
	if err := rc.client.Set(ctx, key, data, rc.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store sighting: %w", err)
	}

	// Add to the active sightings set with timestamp as score
	score := float64(sighting.Timestamp.Unix())
	if err := rc.client.ZAdd(ctx, "active_sightings", redis.Z{
		Score:  score,
		Member: sighting.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add to active sightings: %w", err)
	}

	// Add to geo index for spatial queries
	if err := rc.client.GeoAdd(ctx, "sighting_locations", &redis.GeoLocation{
		Name:      sighting.ID,
		Longitude: sighting.Lng,
		Latitude:  sighting.Lat,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add to geo index: %w", err)
	}

	return nil
}

// migrateSighting migrates old sighting data to new crowdsourced voting fields.
// Applied on deserialization to handle in-flight data from before the migration.
func migrateSighting(s *Sighting) {
	if s.VerificationStatus != "" {
		return // already migrated
	}

	// Migrate counts
	if s.ConfirmationCount > 0 && s.StillThereCount == 0 {
		s.StillThereCount = s.ConfirmationCount
	}
	if s.AllClearCount > 0 && s.NoLongerThereCount == 0 {
		s.NoLongerThereCount = s.AllClearCount
	}

	// Migrate verification status
	switch s.Status {
	case "cleared":
		s.VerificationStatus = "removed"
	case "clearing":
		s.VerificationStatus = "disputed"
	default:
		if s.StillThereCount >= 3 {
			s.VerificationStatus = "community_verified"
		} else {
			s.VerificationStatus = "unverified"
		}
	}

	// Migrate last vote time
	if s.LastVoteAt.IsZero() && !s.LastConfirmedAt.IsZero() {
		s.LastVoteAt = s.LastConfirmedAt
	}
}

// applyStaleness applies lazy staleness demotion on read.
func applyStaleness(s *Sighting) {
	if s.VerificationStatus == "" {
		return
	}

	lastActivity := s.LastVoteAt
	if lastActivity.IsZero() {
		lastActivity = s.Timestamp
	}
	hoursSinceLastVote := time.Since(lastActivity).Hours()
	hoursSinceCreation := time.Since(s.Timestamp).Hours()

	switch s.VerificationStatus {
	case "community_verified":
		if hoursSinceLastVote >= 4 {
			s.VerificationStatus = "unverified"
		}
	case "unverified":
		if hoursSinceCreation >= 8 && hoursSinceLastVote >= 8 {
			s.VerificationStatus = "removed"
		}
	}
}

// GetSighting retrieves a single sighting by ID
func (rc *RedisClient) GetSighting(ctx context.Context, id string) (*Sighting, error) {
	key := fmt.Sprintf("sighting:%s", id)

	data, err := rc.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sighting: %w", err)
	}

	var sighting Sighting
	if err := json.Unmarshal(data, &sighting); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sighting: %w", err)
	}

	migrateSighting(&sighting)
	applyStaleness(&sighting)

	return &sighting, nil
}

// GetActiveSightings retrieves active sightings with optional cursor-based pagination.
// When limit is 0, all sightings are returned (backward compatible).
// When limit > 0, returns at most limit sightings newer-first, with hasMore indicating more pages.
func (rc *RedisClient) GetActiveSightings(ctx context.Context, limit int, before float64) ([]Sighting, bool, error) {
	var ids []string
	var err error

	if limit > 0 {
		// Paginated: fetch newest-first using ZREVRANGEBYSCORE with LIMIT
		// Use "(" prefix for exclusive upper bound to avoid duplicates across pages
		ids, err = rc.client.ZRevRangeByScore(ctx, "active_sightings", &redis.ZRangeBy{
			Min:    "-inf",
			Max:    fmt.Sprintf("(%f", before),
			Offset: 0,
			Count:  int64(limit + 1), // +1 to detect has_more
		}).Result()
	} else {
		// Unpaginated: fetch all newest-first
		ids, err = rc.client.ZRevRange(ctx, "active_sightings", 0, -1).Result()
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to get active sighting IDs: %w", err)
	}

	// Detect has_more and trim
	hasMore := false
	if limit > 0 && len(ids) > limit {
		hasMore = true
		ids = ids[:limit]
	}

	if len(ids) == 0 {
		return []Sighting{}, false, nil
	}

	// Batch fetch all sightings using MGET
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("sighting:%s", id)
	}

	results, err := rc.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, false, fmt.Errorf("failed to batch get sightings: %w", err)
	}

	sightings := make([]Sighting, 0, len(ids))
	var expiredIDs []any

	for i, result := range results {
		if result == nil {
			expiredIDs = append(expiredIDs, ids[i])
			continue
		}

		data, ok := result.(string)
		if !ok {
			continue
		}

		var sighting Sighting
		if err := json.Unmarshal([]byte(data), &sighting); err != nil {
			continue
		}
		migrateSighting(&sighting)
		applyStaleness(&sighting)
		sightings = append(sightings, sighting)
	}

	// Clean up expired entries
	if len(expiredIDs) > 0 {
		rc.client.ZRem(ctx, "active_sightings", expiredIDs...)
		rc.client.ZRem(ctx, "sighting_locations", expiredIDs...)
	}

	return sightings, hasMore, nil
}

// GetNearbySightings retrieves active sightings within a radius of a point
func (rc *RedisClient) GetNearbySightings(ctx context.Context, lat, lng, radiusMiles float64) ([]Sighting, error) {
	locations, err := rc.client.GeoSearch(ctx, "sighting_locations", &redis.GeoSearchQuery{
		Longitude:  lng,
		Latitude:   lat,
		Radius:     radiusMiles,
		RadiusUnit: "mi",
		Sort:       "ASC",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to geo search sightings: %w", err)
	}

	if len(locations) == 0 {
		return []Sighting{}, nil
	}

	// Batch fetch all sightings using MGET
	keys := make([]string, len(locations))
	for i, id := range locations {
		keys[i] = fmt.Sprintf("sighting:%s", id)
	}

	results, err := rc.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to batch get sightings: %w", err)
	}

	sightings := make([]Sighting, 0, len(locations))
	var expiredIDs []any

	for i, result := range results {
		if result == nil {
			expiredIDs = append(expiredIDs, locations[i])
			continue
		}

		data, ok := result.(string)
		if !ok {
			continue
		}

		var sighting Sighting
		if err := json.Unmarshal([]byte(data), &sighting); err != nil {
			continue
		}
		migrateSighting(&sighting)
		applyStaleness(&sighting)
		sightings = append(sightings, sighting)
	}

	// Clean up expired entries from both sets
	if len(expiredIDs) > 0 {
		rc.client.ZRem(ctx, "active_sightings", expiredIDs...)
		rc.client.ZRem(ctx, "sighting_locations", expiredIDs...)
	}

	return sightings, nil
}

// CleanupExpired removes expired sightings from the active set
func (rc *RedisClient) CleanupExpired(ctx context.Context) error {
	ids, err := rc.client.ZRange(ctx, "active_sightings", 0, -1).Result()
	if err != nil {
		return err
	}

	for _, id := range ids {
		key := fmt.Sprintf("sighting:%s", id)
		exists, err := rc.client.Exists(ctx, key).Result()
		if err != nil {
			continue
		}
		if exists == 0 {
			rc.client.ZRem(ctx, "active_sightings", id)
			rc.client.ZRem(ctx, "sighting_locations", id)
		}
	}

	return nil
}

// UpdateSighting updates a sighting in Redis
func (rc *RedisClient) UpdateSighting(ctx context.Context, sighting *Sighting) error {
	key := fmt.Sprintf("sighting:%s", sighting.ID)

	data, err := json.Marshal(sighting)
	if err != nil {
		return fmt.Errorf("failed to marshal sighting: %w", err)
	}

	// Get existing TTL to preserve it
	ttl, err := rc.client.TTL(ctx, key).Result()
	if err != nil || ttl < 0 {
		ttl = rc.ttl
	}

	if err := rc.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to update sighting: %w", err)
	}

	return nil
}

// HasUserAction checks if a user (IP) has already performed an action on a sighting
func (rc *RedisClient) HasUserAction(ctx context.Context, sightingID, ip, actionType string) (bool, error) {
	key := fmt.Sprintf("%s:%s:%s", actionType, sightingID, ip)
	exists, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check user action: %w", err)
	}
	return exists > 0, nil
}

// RecordUserAction records that a user performed an action on a sighting
func (rc *RedisClient) RecordUserAction(ctx context.Context, sightingID, ip, actionType string) error {
	key := fmt.Sprintf("%s:%s:%s", actionType, sightingID, ip)
	// Set with TTL matching sighting TTL
	if err := rc.client.Set(ctx, key, "1", rc.ttl).Err(); err != nil {
		return fmt.Errorf("failed to record user action: %w", err)
	}
	return nil
}

// UpdateSightingWithTTL updates a sighting with a specific TTL
func (rc *RedisClient) UpdateSightingWithTTL(ctx context.Context, sighting *Sighting, ttl time.Duration) error {
	key := fmt.Sprintf("sighting:%s", sighting.ID)

	data, err := json.Marshal(sighting)
	if err != nil {
		return fmt.Errorf("failed to marshal sighting: %w", err)
	}

	if err := rc.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to update sighting: %w", err)
	}

	return nil
}

// HasResponded checks if a responder has already responded to a support request
func (rc *RedisClient) HasResponded(ctx context.Context, sightingID, responderID string) (bool, error) {
	key := fmt.Sprintf("responder:%s:%s", sightingID, responderID)
	exists, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check responder: %w", err)
	}
	return exists > 0, nil
}

// RecordResponder records that a user is responding to a support request
func (rc *RedisClient) RecordResponder(ctx context.Context, sightingID, responderID string, response *SupportResponse) error {
	// Mark responder as responding
	key := fmt.Sprintf("responder:%s:%s", sightingID, responderID)
	if err := rc.client.Set(ctx, key, "1", rc.ttl).Err(); err != nil {
		return fmt.Errorf("failed to record responder: %w", err)
	}

	// Store response details
	responseKey := fmt.Sprintf("response:%s:%s", sightingID, responderID)
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	if err := rc.client.Set(ctx, responseKey, data, rc.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store response: %w", err)
	}

	return nil
}

// StoreFeedback saves feedback to Redis with 30-day TTL
func (rc *RedisClient) StoreFeedback(ctx context.Context, feedback *Feedback) error {
	key := fmt.Sprintf("feedback:%s", feedback.ID)

	data, err := json.Marshal(feedback)
	if err != nil {
		return fmt.Errorf("failed to marshal feedback: %w", err)
	}

	// 30-day TTL for feedback
	ttl := 30 * 24 * time.Hour
	if err := rc.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store feedback: %w", err)
	}

	return nil
}

// checkAndRecordRateLimitScript atomically checks and records a rate limit entry.
// Returns 1 if allowed, 0 if rate-limited.
var checkAndRecordRateLimitScript = redis.NewScript(`
local key = KEYS[1]
local cutoff = tonumber(ARGV[1])
local maxCount = tonumber(ARGV[2])
local nowScore = tonumber(ARGV[3])
local nowMember = ARGV[4]
local windowSeconds = tonumber(ARGV[5])

-- Clean old entries
redis.call('ZREMRANGEBYSCORE', key, '-inf', cutoff)

-- Check count
local count = redis.call('ZCOUNT', key, cutoff, '+inf')
if count >= maxCount then
    return 0
end

-- Record new entry
redis.call('ZADD', key, nowScore, nowMember)
redis.call('EXPIRE', key, windowSeconds)
return 1
`)

// AtomicCheckAndRecordRateLimit atomically checks and records a rate limit.
// Returns true if allowed.
func (rc *RedisClient) AtomicCheckAndRecordRateLimit(ctx context.Context, key string, maxCount int, windowMinutes int) (bool, error) {
	now := time.Now()
	cutoff := float64(now.Add(-time.Duration(windowMinutes) * time.Minute).Unix())

	result, err := checkAndRecordRateLimitScript.Run(ctx, rc.client,
		[]string{key},
		cutoff, maxCount, float64(now.Unix()), fmt.Sprintf("%d", now.UnixNano()), windowMinutes*60*2,
	).Int64()
	if err != nil {
		return false, fmt.Errorf("failed to check/record rate limit: %w", err)
	}

	return result == 1, nil
}

// findOrCreateUserScript atomically looks up or creates a user in Redis.
var findOrCreateUserScript = redis.NewScript(`
local emailKey = KEYS[1]
local userKey = KEYS[2]
local userId = ARGV[1]
local userData = ARGV[2]
local ttl = tonumber(ARGV[3])

local existingUserId = redis.call('GET', emailKey)
if existingUserId then
    local existing = redis.call('GET', 'user:' .. existingUserId)
    return existing
end

redis.call('SET', userKey, userData, 'EX', ttl)
redis.call('SET', emailKey, userId, 'EX', ttl)
return userData
`)

// FindOrCreateUser looks up a user by email hash, creating one if not found (atomic)
func (rc *RedisClient) FindOrCreateUser(ctx context.Context, emailHash string, userTTL time.Duration) (*User, error) {
	emailKey := fmt.Sprintf("email_hash:%s", emailHash)

	now := time.Now()
	newUser := &User{
		ID:            generateUUID(),
		EmailHash:     emailHash,
		EmailVerified: true,
		CreatedAt:     now,
		LastActive:    now,
	}

	userData, err := json.Marshal(newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	userKey := fmt.Sprintf("user:%s", newUser.ID)
	ttlSeconds := int(userTTL.Seconds())

	result, err := findOrCreateUserScript.Run(ctx, rc.client,
		[]string{emailKey, userKey},
		newUser.ID, string(userData), ttlSeconds,
	).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	resultStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result type from Lua script")
	}

	var user User
	if err := json.Unmarshal([]byte(resultStr), &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user from Lua result: %w", err)
	}

	return &user, nil
}

// GetUser retrieves a user by ID
func (rc *RedisClient) GetUser(ctx context.Context, userID string) (*User, error) {
	key := fmt.Sprintf("user:%s", userID)

	data, err := rc.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

// UpdateUserActivity updates the user's last active timestamp and refreshes TTLs
func (rc *RedisClient) UpdateUserActivity(ctx context.Context, user *User, userTTL time.Duration) error {
	user.LastActive = time.Now()

	userData, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	userKey := fmt.Sprintf("user:%s", user.ID)
	if err := rc.client.Set(ctx, userKey, userData, userTTL).Err(); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Refresh email hash TTL
	emailKey := fmt.Sprintf("email_hash:%s", user.EmailHash)
	rc.client.Expire(ctx, emailKey, userTTL)

	return nil
}

// IncrementUserReports increments the user's report counter
func (rc *RedisClient) IncrementUserReports(ctx context.Context, userID string, userTTL time.Duration) error {
	user, err := rc.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	user.ReportsSubmitted++
	user.LastActive = time.Now()

	userData, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	userKey := fmt.Sprintf("user:%s", userID)
	return rc.client.Set(ctx, userKey, userData, userTTL).Err()
}

// StoreSession saves a session in Redis
func (rc *RedisClient) StoreSession(ctx context.Context, jti string, session *Session, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", jti)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return rc.client.Set(ctx, key, data, ttl).Err()
}

// GetSession retrieves a session by JTI
func (rc *RedisClient) GetSession(ctx context.Context, jti string) (*Session, error) {
	key := fmt.Sprintf("session:%s", jti)

	data, err := rc.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// DeleteSession removes a session from Redis (logout)
func (rc *RedisClient) DeleteSession(ctx context.Context, jti string) error {
	key := fmt.Sprintf("session:%s", jti)
	return rc.client.Del(ctx, key).Err()
}

// CheckUserRateLimit checks if a user has exceeded the rate limit (5 reports per 24h)
func (rc *RedisClient) CheckUserRateLimit(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("rate_limit:user:%s", userID)
	cutoff := float64(time.Now().Add(-24 * time.Hour).Unix())

	count, err := rc.client.ZCount(ctx, key, fmt.Sprintf("%f", cutoff), "+inf").Result()
	if err != nil {
		return false, fmt.Errorf("failed to check rate limit: %w", err)
	}

	return count < 5, nil
}

// RecordUserReport records a report timestamp for rate limiting
func (rc *RedisClient) RecordUserReport(ctx context.Context, userID string) error {
	key := fmt.Sprintf("rate_limit:user:%s", userID)
	now := time.Now()

	// Add current timestamp
	if err := rc.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	}).Err(); err != nil {
		return fmt.Errorf("failed to record report: %w", err)
	}

	// Clean up old entries
	cutoff := float64(now.Add(-24 * time.Hour).Unix())
	rc.client.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%f", cutoff))

	// Set key expiry
	rc.client.Expire(ctx, key, 24*time.Hour)

	return nil
}

// CheckIPRateLimit checks if an IP is within its rate limit cooldown
func (rc *RedisClient) CheckIPRateLimit(ctx context.Context, ip string, cooldownMinutes int) (bool, error) {
	key := fmt.Sprintf("rate_limit:ip:%s", ip)
	cutoff := float64(time.Now().Add(-time.Duration(cooldownMinutes) * time.Minute).Unix())

	count, err := rc.client.ZCount(ctx, key, fmt.Sprintf("%f", cutoff), "+inf").Result()
	if err != nil {
		return false, fmt.Errorf("failed to check IP rate limit: %w", err)
	}

	return count < 1, nil
}

// RecordIPReport records a report timestamp for IP-based rate limiting
func (rc *RedisClient) RecordIPReport(ctx context.Context, ip string, cooldownMinutes int) error {
	key := fmt.Sprintf("rate_limit:ip:%s", ip)
	now := time.Now()

	if err := rc.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	}).Err(); err != nil {
		return fmt.Errorf("failed to record IP report: %w", err)
	}

	// Clean up old entries
	cutoff := float64(now.Add(-time.Duration(cooldownMinutes) * time.Minute).Unix())
	rc.client.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%f", cutoff))

	// Set key expiry at 2x cooldown
	rc.client.Expire(ctx, key, time.Duration(cooldownMinutes*2)*time.Minute)

	return nil
}

// CheckActionRateLimit checks if an identity has exceeded the rate limit for an action type.
// Returns (allowed bool, retryAfterSeconds int, error).
func (rc *RedisClient) CheckActionRateLimit(ctx context.Context, identity, action string, maxCount int, windowMinutes int) (bool, int, error) {
	key := fmt.Sprintf("rate_limit:action:%s:%s", action, identity)
	cutoff := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)

	count, err := rc.client.ZCount(ctx, key, fmt.Sprintf("%f", float64(cutoff.Unix())), "+inf").Result()
	if err != nil {
		return false, 0, fmt.Errorf("failed to check action rate limit: %w", err)
	}

	if count >= int64(maxCount) {
		// Calculate retry-after from oldest entry in window
		oldest, err := rc.client.ZRangeWithScores(ctx, key, 0, 0).Result()
		if err == nil && len(oldest) > 0 {
			oldestTime := time.Unix(int64(oldest[0].Score), 0)
			retryAfter := int(time.Duration(windowMinutes)*time.Minute - time.Since(oldestTime))
			if retryAfter < 0 {
				retryAfter = 0
			}
			return false, retryAfter / int(time.Second), nil
		}
		return false, windowMinutes * 60, nil
	}

	return true, 0, nil
}

// RecordAction records an action for rate limiting purposes.
func (rc *RedisClient) RecordAction(ctx context.Context, identity, action string, windowMinutes int) error {
	key := fmt.Sprintf("rate_limit:action:%s:%s", action, identity)
	now := time.Now()

	if err := rc.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	}).Err(); err != nil {
		return fmt.Errorf("failed to record action: %w", err)
	}

	cutoff := float64(now.Add(-time.Duration(windowMinutes) * time.Minute).Unix())
	rc.client.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%f", cutoff))
	rc.client.Expire(ctx, key, time.Duration(windowMinutes*2)*time.Minute)

	return nil
}

// StoreSightingCreator stores the creator identity for a sighting (for withdrawal).
func (rc *RedisClient) StoreSightingCreator(ctx context.Context, sightingID, identity string) error {
	key := fmt.Sprintf("creator:%s", sightingID)
	return rc.client.Set(ctx, key, identity, rc.ttl).Err()
}

// GetSightingCreator retrieves the creator identity for a sighting.
func (rc *RedisClient) GetSightingCreator(ctx context.Context, sightingID string) (string, error) {
	key := fmt.Sprintf("creator:%s", sightingID)
	result, err := rc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get sighting creator: %w", err)
	}
	return result, nil
}

// GetUserVote retrieves the current vote type for a user on a sighting.
// Returns empty string if no vote exists.
func (rc *RedisClient) GetUserVote(ctx context.Context, sightingID, identity string) (string, error) {
	key := fmt.Sprintf("vote:%s:%s", sightingID, identity)
	result, err := rc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user vote: %w", err)
	}
	return result, nil
}

// RecordVote stores or updates a user's vote on a sighting.
func (rc *RedisClient) RecordVote(ctx context.Context, sightingID, identity, voteType string) error {
	key := fmt.Sprintf("vote:%s:%s", sightingID, identity)
	return rc.client.Set(ctx, key, voteType, rc.ttl).Err()
}

// RecordVoteHistory adds a vote entry to the vote history sorted set for decay calculations.
func (rc *RedisClient) RecordVoteHistory(ctx context.Context, sightingID, identity, voteType string) error {
	key := fmt.Sprintf("vote_history:%s", sightingID)
	now := time.Now()
	member := fmt.Sprintf("%s:%s", identity, voteType)
	err := rc.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: member,
	}).Err()
	if err != nil {
		return err
	}
	return rc.client.Expire(ctx, key, rc.ttl).Err()
}

// RemoveVoteHistory removes a specific vote entry from vote history (for vote changes).
func (rc *RedisClient) RemoveVoteHistory(ctx context.Context, sightingID, identity, voteType string) error {
	key := fmt.Sprintf("vote_history:%s", sightingID)
	member := fmt.Sprintf("%s:%s", identity, voteType)
	return rc.client.ZRem(ctx, key, member).Err()
}

// GetVoteHistory retrieves all vote history entries for a sighting.
func (rc *RedisClient) GetVoteHistory(ctx context.Context, sightingID string) ([]VoteEntry, error) {
	key := fmt.Sprintf("vote_history:%s", sightingID)
	results, err := rc.client.ZRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get vote history: %w", err)
	}

	entries := make([]VoteEntry, 0, len(results))
	for _, z := range results {
		member, ok := z.Member.(string)
		if !ok {
			continue
		}
		// member format: "identity:vote_type"
		lastColon := strings.LastIndex(member, ":")
		if lastColon < 0 {
			continue
		}
		identity := member[:lastColon]
		voteType := member[lastColon+1:]
		entries = append(entries, VoteEntry{
			Identity:  identity,
			VoteType:  voteType,
			Timestamp: time.Unix(int64(z.Score), 0),
		})
	}
	return entries, nil
}

// DeleteSighting removes a sighting and its associated keys.
func (rc *RedisClient) DeleteSighting(ctx context.Context, sightingID string) error {
	pipe := rc.client.Pipeline()
	pipe.Del(ctx, fmt.Sprintf("sighting:%s", sightingID))
	pipe.Del(ctx, fmt.Sprintf("creator:%s", sightingID))
	pipe.Del(ctx, fmt.Sprintf("vote_history:%s", sightingID))
	pipe.ZRem(ctx, "active_sightings", sightingID)
	pipe.ZRem(ctx, "sighting_locations", sightingID)
	_, err := pipe.Exec(ctx)
	return err
}

// generateUUID is a helper to generate UUIDs for users
func generateUUID() string {
	return uuid.New().String()
}

// Ping checks if Redis is reachable
func (rc *RedisClient) Ping(ctx context.Context) error {
	return rc.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (rc *RedisClient) Close() error {
	return rc.client.Close()
}
