# Campus Watch (ICE Watch) — AI Context

## Project Overview

Community safety app for reporting ICE sightings near Rutgers University. Anonymous reporting, real-time WebSocket feed, Google OAuth with ScarletMail verification.

## Repository Structure

```
campus-watch/
├── backend/          # Go HTTP server
├── mobile/           # React Native / Expo (SDK 54)
└── CLAUDE.md         # This file
```

---

## Backend (`backend/`)

**Language:** Go 1.21+
**Module:** `campus-watch`
**Dependencies:** `net/http`, `gorilla/websocket`, `go-redis/v9`, standard library

### Key Files

| File | Role |
|---|---|
| `main.go` | Server entry point, route registration, env config |
| `handlers.go` | HTTP handlers; `Handlers` struct holds hub/redis/rateLimiter/authConfig |
| `redis.go` | `RedisClient` — sighting CRUD + auth user/session operations |
| `models.go` | `Sighting`, `SupportResponse`, `Feedback` structs |
| `hub.go` | WebSocket hub — broadcast sightings to connected clients |
| `auth.go` | Google OAuth flow, token exchange, ScarletMail domain check |
| `auth_models.go` | Auth-specific types (User, Session, JWT claims) |
| `auth_handlers.go` | `/auth/*` HTTP handlers |
| `auth_middleware.go` | JWT validation middleware |
| `ratelimit.go` | Per-IP rate limiter |
| `geo.go` | Geospatial helpers |

### Middleware Pattern

```go
func(next http.HandlerFunc) http.HandlerFunc
```

Routes are wrapped with `CORSMiddleware()`.

### Auth System

- Google OAuth + `@scarletmail.rutgers.edu` domain enforcement
- JWT sessions (HMAC-SHA256), stored in Redis
- `AUTH_PHASE` env var: `optional` | `grace` | `enforced`
- Emails are SHA-256 hashed before storage — never plaintext
- Sightings are fully anonymous (no `user_id` field)

### Build & Run

```bash
cd campus-watch/backend
go build ./...
go run .

# With env vars
AUTH_PHASE=optional \
GOOGLE_CLIENT_ID=... \
GOOGLE_CLIENT_SECRET=... \
REDIS_URL=redis://localhost:6379 \
go run .
```

### Required Environment Variables

| Variable | Example | Notes |
|---|---|---|
| `PORT` | `8080` | Defaults to 8080 |
| `REDIS_URL` | `redis://localhost:6379` | |
| `GOOGLE_CLIENT_ID` | `952650426263-...` | Web OAuth client |
| `GOOGLE_CLIENT_SECRET` | `GOCSPX-...` | |
| `JWT_SECRET` | random string | HMAC signing key |
| `AUTH_PHASE` | `optional` | `optional`/`grace`/`enforced` |

---

## Frontend (`mobile/`)

**Framework:** React Native + Expo SDK 54 (managed workflow)
**Language:** JavaScript
**Styling:** NativeWind v4 (Tailwind CSS for RN) + inline styles

### Key Files

| Path | Role |
|---|---|
| `index.js` | App entry point |
| `app.json` | Expo config (bundle ID, permissions, EAS projectId) |
| `eas.json` | EAS Build profiles (development/preview/production) |
| `.env` | Local dev env vars (auto-loaded by Expo) |
| `babel.config.js` | Babel config with NativeWind plugin |
| `metro.config.js` | Metro bundler config with NativeWind |
| `tailwind.config.js` | Tailwind/NativeWind config |
| `global.css` | NativeWind base CSS |

### Source Structure (`src/`)

```
src/
├── contexts/
│   ├── AuthContext.js      # useAuth() hook: login/logout/getAuthHeaders
│   └── SightingContext.js  # Sighting state + WebSocket integration
├── screens/
│   ├── HomeScreen.js
│   ├── MapScreen.js
│   ├── AlertsScreen.js
│   ├── ReportScreen.js
│   ├── RightsScreen.js
│   ├── MoreScreen.js
│   ├── FeedbackScreen.js
│   ├── AuthScreen.js
│   └── OnboardingScreen.js
├── components/
│   ├── AlertCard.js
│   ├── AlertList.js
│   ├── ActivityFeedCard.js
│   ├── ActivityReportForm.js
│   ├── ReportForm.js
│   ├── Map.js / Map.web.js
│   ├── CameraCapture.js
│   ├── VotingButtons.js
│   ├── AnimatedPressable.js
│   └── Settings.js
├── hooks/
│   ├── useWebSocket.js
│   ├── useLocation.js
│   └── useAuthGuard.js
├── services/
│   └── api.js              # API client (reads EXPO_PUBLIC_API_URL)
└── utils/
    ├── constants.js        # COLORS, theme tokens
    └── imageUtils.js
```

### Navigation

```
RootStack (NativeStackNavigator)
└── MainTabs (BottomTabNavigator)
    ├── Home
    ├── Map
    ├── Alerts
    ├── Report
    └── More
```

Auth modal is presented as a screen in `RootStack`.

### Theme

Dark glassmorphism. Defined in `src/utils/constants.js`:
- Background: `#0a0a0f`
- Primary/Amber: `#f59e0b`
- Icons: Lucide React Native

### Run Locally (iOS Simulator)

```bash
cd campus-watch/mobile

# Install deps (if needed)
npm install

# Start Metro + open iOS Simulator
npx expo start --ios

# Or open the Expo Go QR code for device testing
npx expo start
```

Requires Xcode 15+ with iOS Simulator installed.

### EAS Builds

```bash
# Development client (requires EAS account + projectId in app.json)
npx eas build --profile development --platform ios

# Preview build for simulator
npx eas build --profile preview --platform ios
```

Run `npx eas-cli init` once to generate a real `projectId` and update `app.json`.

### Frontend Environment Variables

All prefixed `EXPO_PUBLIC_` (accessible in JS via `process.env.EXPO_PUBLIC_*`).

| Variable | Local Value | Notes |
|---|---|---|
| `EXPO_PUBLIC_API_URL` | `http://localhost:8080` | Backend REST API |
| `EXPO_PUBLIC_WS_URL` | `ws://localhost:8080/ws` | WebSocket endpoint |
| `EXPO_PUBLIC_AUTH_PHASE` | `optional` | Match backend `AUTH_PHASE` |
| `EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID` | `952650426263-l79...` | Google OAuth web client |
| `EXPO_PUBLIC_GOOGLE_IOS_CLIENT_ID` | `952650426263-3pk...` | Google OAuth iOS client |

Local values are stored in `mobile/.env` (gitignored).

---

## Local Development Quick Start

1. Start Redis: `redis-server`
2. Start backend: `cd backend && go run .`
3. Start frontend: `cd mobile && npx expo start --ios`

WebSocket and API calls fail gracefully if the backend is not running (empty state shown).

## Notes

- **Google OAuth on simulator:** The `campuswatch://` redirect scheme must be registered in Google Cloud Console under the iOS OAuth client.
- **Sightings are anonymous:** No user identity is stored with a sighting, only a timestamp and location.
- **`go mod tidy`** after adding any Go dependency.
