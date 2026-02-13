# Report System Analysis

## How the Reporting Protocol Works Today

A user opens the app, grants location permission, and taps the FAB to begin a 4-step modal form: confidence level, category, optional photo, optional description. On submit, the app `POST`s to `/api/sightings` with their GPS coordinates. The backend validates, stores in Redis with a 24h TTL, records a rate limit entry, and broadcasts the sighting over WebSocket to clients within the configured alert radius. Other users can then confirm, report all-clear, or request community support.

---

## Issues With How People May Use the System

### 1. Vote manipulation through IP rotation

Confirmation and all-clear votes are deduplicated by IP address only (`confirmation:{sightingID}:{ip}`). A user with access to a VPN, proxy list, or mobile network toggle can cast unlimited votes from different IPs. Five all-clear votes from one person clears a legitimate sighting. Conversely, hundreds of fake confirmations inflate a false report's credibility.

There is also no rate limit on the `/api/sightings/confirm` or `/api/sightings/all-clear` endpoints at all -- only sighting *creation* is rate-limited.

### 2. Shared networks collapse into one voter

The opposite problem: everyone on the same campus Wi-Fi, library, or dorm network shares a single public IP. If one person confirms a sighting, nobody else on that network can. At a university, this makes the voting system nearly useless for the majority of users who are on `eduroam`.

### 3. False reports persist for 24 hours with no recourse

Once submitted, a sighting cannot be edited, corrected, or withdrawn. If someone misidentifies a situation or submits at the wrong location, the report stays visible for the full 24-hour TTL. The only removal path is community all-clear votes reaching the threshold of 5 -- which itself can be gamed.

### 4. Support request spam

`/api/sightings/request-support` has no rate limit and no authentication requirement (in optional/grace auth phases). Each call broadcasts a high-priority SOS to every client within 2 miles. A single user could call this endpoint in a loop, flooding nearby users with notifications and filling WebSocket send buffers until slow clients get disconnected.

### 5. Fake responders trivially satisfy support thresholds

The respond-to-support endpoint accepts any `responder_id` from the client body with no verification. One person can POST three times with different IDs (`"user1"`, `"user2"`, `"user3"`), satisfy `RespondersNeeded = 3`, and flip the sighting back to "active" -- canceling the support request before real help arrives.

### 6. Race condition in sighting creation rate limiting

Rate limits are checked *before* storage but recorded *after* storage (`handlers.go:136` then `handlers.go:155`). If multiple requests from the same IP arrive simultaneously, all pass `CheckIPRateLimit` (count is still 0) before any `RecordIPReport` executes. In practice this means 3-4 duplicate sightings can slip through per rate-limit window.

### 7. All-clear votes can suppress real threats

The threshold is hardcoded at 5 for every sighting regardless of how many people confirmed it. A sighting with 50 confirmations and 5 all-clears still gets marked "cleared." There is no ratio or quorum logic -- the system treats 5 all-clears as absolute truth regardless of evidence to the contrary.

### 8. "Clearing" state has no timeout

Once a sighting reaches 2 all-clear votes it transitions to "clearing" status. If it never reaches 5, it stays in "clearing" indefinitely (up to the 24h TTL). There is no mechanism to revert to "active" if new confirmations come in, creating a confusing zombie state.

### 9. Unauthenticated mode undermines all per-user protections

In the default `optional` auth phase, anyone can submit reports without authenticating. The IP-based fallback allows 1 report per 5-minute cooldown -- but combined with issue #1, this is trivially bypassed. User-based rate limiting (5 reports/24h) only applies to authenticated users, so the strictest limits apply only to the most cooperative users.

### 10. No feedback loop on rate limits

When rate-limited, users see "Please wait before submitting another report" with no indication of how long to wait. There is no countdown, no `Retry-After` header, and no remaining-quota metadata. Users are left guessing and repeatedly tapping "Try Again."

---

## Design Decisions for a Better Experience

### A. Tie votes to authenticated identity, not IP

Require authentication for confirm/all-clear/support actions (even if reporting stays optional). Deduplicate by `userID` instead of IP. This eliminates both the shared-network problem and the IP-rotation abuse vector in one change. Users on the same Wi-Fi can all vote independently; one person with a VPN gets one vote.

### B. Add rate limits to all write endpoints

Apply per-user (or per-IP) rate limits to confirm, all-clear, request-support, and respond endpoints -- not just sighting creation. A sorted-set pattern identical to the existing one works. Even modest limits (e.g., 10 votes/minute) prevent automated spam while being invisible to normal users.

### C. Use ratio-based clearing instead of absolute threshold

Replace the flat threshold with a formula that considers both sides:

```
cleared = all_clear_count >= min_threshold AND
          all_clear_count / (all_clear_count + confirmation_count) > 0.6
```

This means a sighting with 50 confirmations needs ~75 all-clears rather than 5. Adjust the ratio and minimum threshold based on real-world usage data.

### D. Make rate limit recording atomic with storage

Move `RecordIPReport` / `RecordUserReport` into a Lua script that runs atomically with `StoreSighting`, or check-and-record the rate limit *before* storage using an atomic increment. This closes the race window where concurrent requests all pass the check.

### E. Allow report withdrawal by the original reporter

Add a `DELETE /api/sightings?id=X` endpoint that requires the request to come from the same IP (or authenticated user) that created the sighting. Set a short grace period (e.g., 5 minutes) during which the creator can delete a mistaken report. After the grace period, only community all-clear can remove it.

### F. Return rate limit metadata in responses

Include `Retry-After` (seconds) and `X-RateLimit-Remaining` headers on 429 responses. On the frontend, display a countdown timer instead of a static error message. This turns a frustrating dead-end into a clear expectation.

### G. Validate responder identity

Require authentication for support responses, or at minimum tie `responder_id` to the IP/session so one client can only respond once. The current client-generated anonymous ID (`anon_{timestamp}_{random}`) provides zero deduplication guarantees.

### H. Add a "clearing" timeout with reactivation

If a sighting is in "clearing" status and receives a new confirmation, revert it to "active" and reset `AllClearCount` to 0 (or reduce by the confirmation weight). Additionally, if no new all-clear votes arrive within 30 minutes, automatically revert to "active." This prevents zombie sightings and makes the system responsive to changing conditions.

### I. Require authentication in production

The `optional` auth phase makes sense during initial rollout but should be treated as temporary. The `enforced` phase with ScarletMail verification is the only configuration that provides meaningful per-user accountability. Plan a migration timeline: `optional` (launch week) -> `grace` with in-app nudges (weeks 2-4) -> `enforced` (month 2+).

### J. Let users provide ETA when responding to support

The current hardcoded 5-minute ETA is meaningless. Add an ETA selector (1, 3, 5, 10, 15 min) to the response flow. Display arriving responders and their ETAs on the support-requested card so the person in need has real situational awareness. Optionally add navigation integration to route responders to the location.

---

## Summary of Severity

| Issue | Exploitability | User Impact | Fix Complexity |
|-------|---------------|-------------|----------------|
| Vote manipulation via IP rotation | Easy | High -- undermines trust in entire system | Medium (requires auth for votes) |
| Shared network = one voter | Automatic | High -- most campus users affected | Medium (same fix: auth-based dedup) |
| False reports persist 24h | Passive | Medium -- causes unnecessary fear | Low (add withdrawal endpoint) |
| Support request spam | Easy | High -- floods nearby users | Low (add rate limit) |
| Fake responders | Easy | High -- cancels real SOS requests | Medium (require auth or IP dedup) |
| Creation rate limit race | Requires concurrency | Low-Medium -- extra duplicates | Medium (atomic Lua script) |
| Flat all-clear threshold | Requires coordination | High -- suppresses real threats | Low (ratio formula) |
| Clearing state zombie | Passive | Low -- confusing UI state | Low (add timeout/revert logic) |
| No rate limit feedback | Passive | Medium -- frustrating UX | Low (add headers + countdown) |
| Optional auth in production | Structural | High -- no accountability | Low (config change + migration) |

The highest-impact changes are **auth-based vote deduplication** (fixes #1, #2, #5 simultaneously) and **rate limiting all write endpoints** (fixes #4, partially #6). These two changes alone address 5 of the 10 identified issues.
