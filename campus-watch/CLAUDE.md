# Campus Watch - Claude Code Context

<!-- LIVING DOCUMENT: Update this file after every significant change -->
<!-- This file is the primary context source for Claude Code sessions -->

## Last Updated
- **Date:** 2026-02-11
- **Session:** Initial comprehensive documentation
- **Branch:** master (no commits yet, main branch is `main`)
- **Status:** Full codebase documented, no git history exists yet

## Recent Changes Log
<!-- Newest first. Add entries after every significant change. -->
| Date | Change | Files Modified | Notes |
|------|--------|----------------|-------|
| 2026-02-11 | Initial CLAUDE.md creation | CLAUDE.md | Full codebase audit |

## Project Purpose

Campus Watch is a community safety app for Rutgers University New Brunswick students. It enables real-time, anonymous reporting and tracking of ICE activity on and around campus. The app prioritizes privacy (hashed emails, anonymous reports), community validation (confirm/all-clear voting), and mutual aid (support requests with responder coordination).

## Repository Layout

```
campus-watch/                         # ROOT: /Users/ahmedmohamed/Documents/Melt/campus-watch/
├── backend/                          # Go API server (11 .go files)
│   ├── main.go                       # Entry point, routes, env config, graceful shutdown
│   ├── handlers.go                   # HTTP handlers, scoring, verification state machine (~29KB, largest)
│   ├── models.go                     # All data structs: Sighting, VoteEntry, SupportResponse, etc.
│   ├── redis.go                      # Redis persistence: sighting CRUD, auth, voting, rate limits, Lua scripts (~26KB)
│   ├── hub.go                        # WebSocket hub: client mgmt, geo-filtered broadcasting
│   ├── geo.go                        # Haversine distance calculations
│   ├── ratelimit.go                  # IP extraction with proxy trust support
│   ├── auth.go                       # Google OAuth verification, JWT generation/validation, AuthConfig
│   ├── auth_models.go                # User, Session, GoogleTokenPayload, SessionClaims structs
│   ├── auth_handlers.go              # Login, logout, verify, status endpoints
│   ├── auth_middleware.go            # OptionalAuth, RequireAuth, ActionRateLimit, voter identity
│   ├── go.mod                        # Module: campus-watch, Go 1.21
│   └── go.sum
│
├── mobile/                           # React Native / Expo frontend (28 JS files)
│   ├── App.js                        # Root: SafeAreaProvider > AuthProvider > NavigationContainer > RootStack
│   ├── index.js                      # Entry point
│   ├── app.json                      # Expo SDK 54, bundle ID: com.campuswatch.app
│   ├── package.json                  # React 19.1, RN 0.81.5, Expo 54.0.33
│   ├── src/
│   │   ├── screens/
│   │   │   ├── index.js              # Screen exports
│   │   │   ├── HomeScreen.js         # Activity feed (main view) with stats bar
│   │   │   ├── MapScreen.js          # Interactive map with filter tabs
│   │   │   ├── RightsScreen.js       # Know Your Rights legal guide
│   │   │   ├── MoreScreen.js         # Settings, profile, about (version "2.0.0")
│   │   │   ├── FeedbackScreen.js     # Feedback form (bug/feature/general/privacy)
│   │   │   └── AuthScreen.js         # Google OAuth login modal
│   │   ├── components/
│   │   │   ├── index.js              # Component exports
│   │   │   ├── Icons.js              # 20+ custom View-based icons (no icon library)
│   │   │   ├── Map.js                # react-native-maps wrapper with custom markers
│   │   │   ├── Map.web.js            # Web: react-leaflet variant
│   │   │   ├── AlertCard.js          # Sighting card with confirm/all-clear/support actions
│   │   │   ├── ActivityFeedCard.js   # Feed item with VotingButtons + support
│   │   │   ├── AlertList.js          # Modal list sorted by distance
│   │   │   ├── ReportForm.js         # Single-screen report wizard
│   │   │   ├── ReportForm.web.js     # Web variant
│   │   │   ├── ActivityReportForm.js # Alternative report form (used in HomeScreen)
│   │   │   ├── ActivityReportForm.web.js
│   │   │   ├── CameraCapture.js      # Photo capture/picker with compression
│   │   │   ├── VotingButtons.js      # Still There / No Longer There buttons
│   │   │   └── Settings.js           # Modal settings panel
│   │   ├── contexts/
│   │   │   └── AuthContext.js         # Google OAuth + JWT state, AsyncStorage persistence
│   │   ├── hooks/
│   │   │   ├── useLocation.js         # Expo Location with watch position
│   │   │   ├── useWebSocket.js        # WS connection, reconnect, sighting state
│   │   │   └── useAuthGuard.js        # Auth-gated action wrapper
│   │   ├── services/
│   │   │   └── api.js                 # Centralized API client with auth headers
│   │   └── utils/
│   │       ├── constants.js           # COLORS, API_URL, WS_URL, helpers, categories
│   │       └── imageUtils.js          # Image compression utilities
│   └── assets/                        # Icons, splash screen, favicon
│
├── docker-compose.yml                 # Redis 7 + Backend services
├── Dockerfile.backend                 # Multi-stage: golang:1.21-alpine -> alpine:3.19
├── .env.example                       # Example env vars (uses VITE_ prefix - may be outdated)
├── .gitignore                         # Standard Node/Go ignores
├── REPORT_SYSTEM_ANALYSIS.md          # Security analysis: 10 issues + fixes (many already implemented)
└── CLAUDE.md                          # THIS FILE
```

## Tech Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Backend | Go | 1.21 |
| HTTP | net/http (stdlib) | - |
| WebSocket | gorilla/websocket | 1.5.1 |
| Database | Redis | 7 (Alpine) |
| Redis Client | go-redis/v9 | 9.4.0 |
| Auth | golang-jwt/v5 | 5.2.1 |
| UUIDs | google/uuid | 1.6.0 |
| Frontend | React Native + Expo | SDK 54, RN 0.81.5, React 19.1 |
| Maps (native) | react-native-maps | 1.20.1 |
| Maps (web) | react-leaflet + leaflet | 5.0.0 / 1.9.4 |
| Location | expo-location | 19.0.8 |
| Camera | expo-camera + expo-image-picker | 17.0.10 |
| Auth (frontend) | expo-auth-session | 7.0.10 |
| Storage | @react-native-async-storage | 2.2.0 |
| Notifications | expo-notifications | 0.32.16 (NOT wired up) |
| Deploy | Docker + Docker Compose | - |

## Architecture Decisions

### Why Redis (no SQL database)?
All data is intentionally ephemeral. Sightings expire after 24 hours. User records expire after 30 days of inactivity. Sessions expire after 7 days. This matches the app's privacy-first design -- we don't want a permanent record of who reported what.

### Why anonymous sightings?
The `Sighting` struct has no `user_id` field by design. Reports cannot be traced back to users. A separate `creator:{sightingID}` key allows 5-minute withdrawal but auto-expires.

### Why email hashing?
User emails are SHA-256 hashed before storage. The backend never stores plaintext emails. This verifies Rutgers affiliation without creating a user directory.

### Auth phases (AUTH_PHASE env var)
- `optional` (current default): Auth available but not required. IP-based fallback for rate limiting/vote dedup.
- `grace`: Same as optional but with UI nudges to sign in.
- `enforced`: Auth required for sighting submission. User-based rate limits only.

### WebSocket geo-filtering
New sightings broadcast only to clients within `ALERT_RADIUS_MILES` (default 1 mile). Status updates (confirmations, all-clears) broadcast to ALL connected clients. Support requests broadcast within 2 miles.

### Community validation model
- **Confirm (Still There):** Increments count, deduplicated per user/IP
- **All Clear (No Longer There):** Uses ratio-based clearing: `all_clear / (all_clear + confirmation) > 0.6` AND minimum count >= threshold
- **Support Request:** SOS broadcast to nearby users with responder coordination and ETA tracking

---

## Backend Architecture (Detail)

### Handler Pattern
All handlers are methods on `Handlers` struct:
```go
type Handlers struct {
    hub              *Hub
    redis            *RedisClient
    authConfig       *AuthConfig
    rateLimitMinutes int
}
```
Middleware signature: `func(next http.HandlerFunc) http.HandlerFunc`
Routes wrapped with `CORSMiddleware()`.

### Key Structs

**Sighting** (models.go) - The core data structure:
```
ID, Description, ImageData (base64 max 1MB), Lat, Lng, Confidence (confirmed/likely/uncertain),
Timestamp, ExpiresAt, Category (vehicle/on_foot/other), VehicleType, ApproxCount, MovementDirection,
StillThereCount, NoLongerThereCount, LastVoteAt, VerificationStatus (unverified/community_verified/disputed/removed),
ConfidenceScore (0.0-1.0 decay-weighted), FeedScore, Status (active/support_requested),
SupportRequested, SupportRequestedAt, RespondersCount, RespondersNeeded (default 3)
+ Legacy fields: ConfirmationCount, LastConfirmedAt, AllClearCount, AllClearThreshold
```

**Auth structs** (auth_models.go):
- `User` - ID, EmailHash (SHA-256), EmailVerified, CreatedAt, LastActive, ReportsSubmitted
- `Session` - Token (JWT), UserID, ExpiresAt, DeviceInfo, CreatedAt
- `AuthConfig` - GoogleClientID, JWTSecret, AllowedEmailDomain, SessionDurationHours, UserTTLDays, AuthPhase
- `SessionClaims` - UserID + jwt.RegisteredClaims

**WebSocket types** (models.go):
- `Client` - Hub, Conn, Lat, Lng, Send channel
- `LocationUpdate` - Type ("location"), Lat, Lng
- `WebSocketMessage` - Type ("sighting"), Payload (Sighting)
- `WebSocketUpdateMessage` - Type ("sighting_update"), Payload (SightingUpdate)
- `SupportRequestMessage` - Type ("support_request"), Sighting, Radius

**Voting** (models.go):
- `VoteRequest` - Type (still_there/no_longer_there)
- `VoteEntry` - Identity, VoteType, Timestamp

### Voting & Verification Algorithms (handlers.go)

**Confidence Score** (decay-weighted):
```
weight = exp(-0.1 * hours_since_vote)
score = weightedStillThere / (weightedStillThere + weightedNoLongerThere)
```

**Verification State Machine:**
```
unverified -> community_verified   (if stillThere >= 3 AND score >= 0.7)
community_verified -> disputed     (if score < 0.7)
disputed -> community_verified     (if score >= 0.7)
disputed -> removed                (if noLongerThere >= 3 AND score < 0.3)
removed -> disputed                (if score >= 0.3)
Any -> removed                     (if noLongerThere >= 3 AND score < 0.3)
```

**Staleness demotion** (applied on every read):
- `community_verified` -> `unverified` after 4+ hours without vote
- `unverified` -> `removed` after 8+ hours since creation AND 8+ hours without vote

**Feed Scoring:**
```
feedScore = (statusWeight * 40) + (confidenceScore * 20) + (voteRecency * 25) + (timeDecay * 10)

statusWeight: support_requested=1.0, community_verified=0.8, unverified=0.6, disputed=0.3, removed=0.1
voteRecency: exp(-0.2 * hoursSinceLastVote)
timeDecay: exp(-0.05 * hoursSinceCreation)
```

### Rate Limiting

| Endpoint | Limit | Window | Identity |
|----------|-------|--------|----------|
| Create sighting (auth) | 5 | 24h | user:{userID} |
| Create sighting (no auth) | 1 | RATE_LIMIT_MINUTES | ip:{clientIP} |
| Vote (confirm/all-clear) | 20 | 5 min | user or ip |
| Request support | 3 | 10 min | user or ip |
| Respond to support | 5 | 5 min | user or ip |

Atomic rate limiting via Lua scripts in Redis. Action-specific limits via `ActionRateLimitMiddleware`.

### Redis Key Patterns

| Key | Purpose | TTL |
|-----|---------|-----|
| `sighting:{id}` | Full sighting JSON | SIGHTING_TTL_HOURS (24h) |
| `active_sightings` | Sorted set of IDs (score=timestamp) | - |
| `sighting_locations` | Geo index for spatial queries | - |
| `vote:{sightingID}:{identity}` | Current vote type | 24h |
| `vote_history:{sightingID}` | Sorted set of votes (score=timestamp) | 24h |
| `creator:{sightingID}` | Original creator identity | 5 min (withdrawal grace) |
| `responder:{sightingID}:{identity}` | Responded marker | 24h |
| `response:{sightingID}:{identity}` | Response detail JSON (ETA) | 24h |
| `rate_limit:ip:{ip}` | Report timestamps sorted set | 2x cooldown |
| `rate_limit:user:{userID}` | Report timestamps sorted set | 24h |
| `rate_limit:action:{action}:{identity}` | Action timestamps sorted set | 2x window |
| `feedback:{id}` | Feedback JSON | 30d |
| `user:{userID}` | User JSON | USER_TTL_DAYS (30d) |
| `email_hash:{emailHash}` | Maps hash -> userID | 30d |
| `session:{jti}` | Session JSON | SESSION_DURATION_HOURS (7d) |

---

## Frontend Architecture (Detail)

### Navigation Structure
```
SafeAreaProvider > AuthProvider > NavigationContainer > RootStack (NativeStackNavigator)
  ├── Main > MainTabs (BottomTabNavigator)
  │   ├── Home -> HomeScreen (activity feed, default tab)
  │   ├── Map -> MapScreen (interactive map + filter tabs)
  │   ├── Rights -> RightsScreen (legal guide)
  │   └── More -> MoreStackNav (nested stack)
  │       ├── MoreMain -> MoreScreen (settings/profile)
  │       └── Feedback -> FeedbackScreen
  └── Auth (modal presentation) -> AuthScreen
```

### State Management
- **No Redux.** Uses React Context + hooks + AsyncStorage.
- `AuthContext`: Global auth state (user, token, isAuthenticated, isLoading)
- `useWebSocket`: Real-time sighting state (connects, reconnects, manages sighting array)
- `useLocation`: GPS position tracking (watch with 10s/10m threshold)
- `useAuthGuard`: Auth-gated action wrapper (redirects in enforced mode)

### API Service Layer (services/api.js)
Centralized client with automatic auth header injection:
```javascript
api.getSightings()
api.createSighting(data)
api.deleteSighting(id)
api.confirmSighting(id)
api.allClearSighting(id)
api.vote(id, type)                    // type: 'still_there' | 'no_longer_there'
api.requestSupport(id)
api.respondToSupport(id, { responder_id, eta })
api.cancelSupport(id)
api.submitFeedback({ type, message, device_info })
```
Handles 429 (rate limit with Retry-After), 409 (conflict/duplicate), throws `ApiError`.

### AsyncStorage Keys
- `campus_watch_auth_token` - JWT
- `campus_watch_auth_expiry` - Expiry timestamp
- `campus_watch_auth_user_id` - User ID
- `campus_watch_notification_mode` - normal/vibration/silent
- `campus_watch_alert_radius` - miles (0.5/1/2/5)

### WebSocket Protocol
```
Connect: WS_URL (default ws://localhost:8080/ws)
Client sends: { type: 'location', lat: number, lng: number }  // every 30s
Server sends: { type: 'sighting', payload: Sighting }          // new sighting (geo-filtered)
Server sends: { type: 'sighting_update', payload: SightingUpdate }  // vote update (all clients)
Server sends: { type: 'support_request', sighting: Sighting, radius: number }  // SOS (2mi radius)
```
Reconnection: exponential backoff (1s, 2s, 4s... max 30s). Re-fetches via REST on reconnect.

### Theme (constants.js)
Light theme with red accent:
- Background: `#ffffff`
- Text: `#0f172a`
- Accent: `#d4183d`
- Muted: `#94a3b8`
- Success: `#16a34a`
- Danger: `#dc2626`

Default map center: Rutgers NB (40.5008, -74.4474)

---

## API Routes Quick Reference

```
GET    /health                           CORSMiddleware(HealthHandler)
GET    /api/sightings                    CORSMiddleware(GetSightingsHandler)           ?lat=&lng=&radius= geo filter
POST   /api/sightings                    CORSMiddleware(sightingPostHandler)            auth-phase dependent middleware chain
DELETE /api/sightings                    CORSMiddleware(OptionalAuth(DeleteHandler))    ?id=  creator only, 5-min grace
POST   /api/sightings/vote              CORSMiddleware(OptionalAuth(ActionRL("vote",20,5,VoteHandler)))
POST   /api/sightings/confirm           CORSMiddleware(OptionalAuth(ActionRL("vote",20,5,ConfirmHandler)))     legacy
POST   /api/sightings/all-clear         CORSMiddleware(OptionalAuth(ActionRL("vote",20,5,AllClearHandler)))    legacy
POST   /api/sightings/request-support   CORSMiddleware(OptionalAuth(ActionRL("request_support",3,10,RequestSupportHandler)))
POST   /api/sightings/respond           CORSMiddleware(OptionalAuth(ActionRL("respond",5,5,RespondToSupportHandler)))  ?id=
POST   /api/sightings/cancel-support    CORSMiddleware(CancelSupportHandler)           ?id=
POST   /api/feedback                    CORSMiddleware(FeedbackHandler)
POST   /auth/google/login               CORSMiddleware(GoogleLoginHandler)
GET    /auth/status                      CORSMiddleware(OptionalAuth(AuthStatusHandler))
POST   /auth/verify                      CORSMiddleware(VerifyHandler)
POST   /auth/logout                      CORSMiddleware(LogoutHandler)
WS     /ws                               WebSocketHandler (unauthenticated)
```

## Environment Variables

### Backend
| Variable | Default | Notes |
|----------|---------|-------|
| `PORT` | `8080` | Server listen port |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection |
| `ALERT_RADIUS_MILES` | `1.0` | WebSocket broadcast radius |
| `RATE_LIMIT_MINUTES` | `5` | IP-based cooldown |
| `SIGHTING_TTL_HOURS` | `24` | Sighting expiry (docker-compose uses 3) |
| `GOOGLE_CLIENT_ID` | *(required for auth)* | Google OAuth client ID |
| `JWT_SECRET` | *(required for auth)* | HMAC-SHA256 signing key |
| `ALLOWED_EMAIL_DOMAIN` | `scarletmail.rutgers.edu` | Restricted email domain |
| `AUTH_PHASE` | `optional` | optional / grace / enforced |
| `SESSION_DURATION_HOURS` | `168` (7 days) | JWT expiration |
| `USER_TTL_DAYS` | `30` | User record TTL |
| `ALLOWED_ORIGIN` | `*` | CORS origin |
| `TRUST_PROXY` | `false` | Trust X-Forwarded-For |

### Frontend (EXPO_PUBLIC_ prefix)
| Variable | Default |
|----------|---------|
| `EXPO_PUBLIC_API_URL` | `http://localhost:8080` |
| `EXPO_PUBLIC_WS_URL` | `ws://localhost:8080/ws` |
| `EXPO_PUBLIC_AUTH_PHASE` | `optional` |
| `EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID` | - |
| `EXPO_PUBLIC_GOOGLE_IOS_CLIENT_ID` | - |
| `EXPO_PUBLIC_GOOGLE_ANDROID_CLIENT_ID` | - |

## Building and Running

```bash
# Backend (from repo root)
cd campus-watch/backend && go run .
# or: go build -o server . && ./server

# Frontend (from repo root)
cd campus-watch/mobile && npx expo start

# Redis (Docker)
docker compose up redis

# Full stack (Docker)
docker compose up

# After adding Go dependencies
cd campus-watch/backend && go mod tidy
```

## Known Gaps / Not Yet Implemented

### High Priority
- **Push notifications:** `expo-notifications` is in package.json but NOT wired up. Users only get alerts with active WebSocket.
- **Vehicle details in UI:** Backend collects `vehicle_type`, `approx_count`, `movement_direction` but ReportForm doesn't expose input fields for them.
- **WebSocket not authenticated:** Anyone can connect to `/ws` and receive broadcasts.
- **No automated tests:** Zero test files for backend or frontend.

### Medium Priority
- **No structured logging:** Backend uses `log.Printf` everywhere.
- **Background location:** Permissions configured in app.json but not implemented in code.
- **No offline support:** No caching of sightings or offline report queue.
- **No error boundary component** in frontend.
- **No loading skeleton screens.**

### Low Priority
- **No admin/moderation tools.**
- **No i18n/localization.**
- **Auth defaults to optional:** Vote manipulation possible via IP rotation in optional mode.
- **docker-compose SIGHTING_TTL_HOURS=3** differs from default of 24h.
- **.env.example uses VITE_ prefix** instead of EXPO_PUBLIC_ (outdated).

## Implemented Fixes from REPORT_SYSTEM_ANALYSIS.md

The REPORT_SYSTEM_ANALYSIS.md identified 10 issues. Here's what has been addressed:

| Issue | Status | Implementation |
|-------|--------|----------------|
| Vote manipulation via IP rotation | Partially fixed | Auth-based identity (`user:{id}` or `ip:{ip}`) for vote dedup |
| Shared network = one voter | Partially fixed | Auth users get individual identity regardless of IP |
| False reports persist 24h | Fixed | DELETE endpoint with 5-min creator grace period |
| Support request spam | Fixed | ActionRateLimitMiddleware (3 per 10 min) |
| Fake responders | Partially fixed | Server-derived identity (user/IP) for dedup, not client-generated |
| Rate limit race condition | Fixed | Lua script `AtomicCheckAndRecordRateLimit` |
| Flat all-clear threshold | Fixed | Ratio-based verification with decay-weighted confidence scoring |
| Clearing state zombie | Fixed | Staleness demotion on read (4h/8h thresholds) |
| No rate limit feedback | Partially fixed | 429 responses include retry info from `CheckActionRateLimit` |
| Optional auth in production | Not fixed | Still defaults to optional; needs migration plan |

## Gotchas / Common Pitfalls

1. **Go module name is `campus-watch`** (with hyphen). Imports are `campus-watch/...`.
2. **Build from `backend/` directory**, not repo root: `cd campus-watch/backend && go run .`
3. **Frontend deps** are in `mobile/package.json`, not root.
4. **No git commits exist yet.** The repo is freshly initialized on `master` with `main` as the target PR branch.
5. **No git remote configured.** Push will need a remote added first.
6. **Legacy endpoints still exist:** `/api/sightings/confirm` and `/api/sightings/all-clear` map to the new vote system internally. The canonical endpoint is `/api/sightings/vote` with `{ type: "still_there" | "no_longer_there" }`.
7. **Sighting migration on read:** Old fields (`ConfirmationCount`, `AllClearCount`, status `"cleared"/"clearing"`) are automatically migrated to new fields on deserialization.
8. **Image data stripped from WebSocket broadcasts** to reduce bandwidth. Clients must fetch full sighting via REST for images.
9. **WebSocket hub constants:** `maxMessageSize = 512 bytes` (location updates only), `pingPeriod ~54s`, `pongWait 60s`.
10. **Support request broadcasts at 2-mile radius**, overriding the default `ALERT_RADIUS_MILES`.
