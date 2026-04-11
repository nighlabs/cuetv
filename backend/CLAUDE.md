# CueTV Backend Developer

You are the backend developer for CueTV. This file extends the root `CLAUDE.md` — read that first for project overview, architectural rules, git workflow, and security requirements. This file covers everything specific to the `backend/` directory.

---

## Stack

- **Language:** Go with Chi router
- **Database:** SQLite via sqlc (type-safe generated queries)
- **Auth:** JWT (admin tokens + friend tokens), admin portal password via env var
- **Real-time:** SSE broker (viewer playback), WebSocket hub (admin queue sync)
- **Deployment:** Docker container, SQLite persisted to named volume at `/data/cuetv.db`

---

## Directory Structure

```
backend/
├── cmd/server/           # main.go — wire dependencies, start server
└── internal/
    ├── broker/           # SSE broker + WebSocket hub
    ├── config/           # Env var loading and validation
    ├── database/         # SQLite connection, migration runner
    ├── db/               # sqlc-generated — NEVER manually edit
    ├── handlers/         # HTTP handlers — thin, delegate to services
    ├── middleware/        # JWT auth, CORS, rate limiting, request logging
    ├── models/           # Request/response DTOs
    └── services/         # All business logic
```

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | Yes | — | JWT signing secret |
| `ADMIN_PORTAL_PASSWORD` | Yes | — | Admin portal access password |
| `DATABASE_PATH` | No | `/data/cuetv.db` | SQLite file path |
| `PORT` | No | `8080` | HTTP server port |
| `ADMIN_TOKEN_DURATION` | No | `168h` | Admin JWT validity |
| `FRIEND_TOKEN_DURATION` | No | `12h` | Friend JWT validity |
| `RATE_LIMIT_PER_MINUTE` | No | `10` | Rate limit per IP |
| `TRUSTED_PROXIES` | No | — | Comma-separated CIDRs for trusted proxies |
| `SENTRY_DSN` | No | — | Sentry DSN for error tracking |

Config must be validated at startup — if a required env var is missing, exit with a clear error. Never start in a broken state.

---

## API Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/health` | None | Health check |
| POST | `/api/admin/verify` | None | Verify admin portal password |
| POST | `/api/sessions` | Admin password | Create session → JWT, friend key, viewer URL |
| POST | `/api/sessions/join` | None | Join with friend key → JWT |
| POST | `/api/sessions/rejoin` | JWT | Rejoin as admin |
| GET | `/api/sessions/:id` | JWT | Session details |
| GET | `/api/sessions/:id/events` | Viewer token (query param) | SSE stream — viewer playback commands |
| GET | `/api/sessions/:id/queue-events` | JWT (header) | WebSocket — admin queue sync |
| GET | `/api/sessions/:id/queue` | JWT | Queue state |
| POST | `/api/sessions/:id/queue` | JWT | Add video `{ url, marqueeText }` |
| PATCH | `/api/sessions/:id/queue` | JWT | Reorder `{ order: []string }` (item IDs) |
| DELETE | `/api/sessions/:id/queue/:itemId` | JWT | Remove item |
| POST | `/api/sessions/:id/playback` | JWT | `{ command: "play"\|"pause"\|"next"\|"prev" }` |
| POST | `/api/sessions/:id/video-ended` | Viewer token | Signal video end — triggers auto-advance |
| GET | `/api/sessions/:id/config` | JWT | Room config |
| PATCH | `/api/sessions/:id/config` | JWT | Update room config |

---

## SSE Broker

- Each session has its own subscriber list — never cross-contaminate sessions
- On connection: register subscriber; on disconnect: unregister and clean up
- Event format: `play`, `pause`, `load:{videoId}`, `queue:ended`
- Use buffered channels — drop events for slow consumers rather than blocking the broadcaster
- Log connection count per session on register/unregister

---

## WebSocket Hub

- Each session has its own hub
- Validate JWT before completing the WebSocket upgrade — reject with 401 if invalid
- Broadcast `queue:updated` and `config:updated` JSON messages to all connected admins for a session
- Handle disconnects cleanly — do not panic on closed connections

---

## Auto-Advance Logic

When `POST /api/sessions/:id/video-ended` is received:

1. **Debounce:** if auto-advance was triggered for this session within the last 3 seconds, ignore the signal
2. Increment `currentIndex` in the DB
3. Look up the queue item at the new index
4. If found: broadcast `load:{videoId}` via SSE to all viewers + `queue:updated` via WebSocket to all admins
5. If not found (end of queue): broadcast `queue:ended` via SSE; do not advance

---

## YouTube URL Parsing

Handle all valid YouTube URL formats:

- `https://www.youtube.com/watch?v=VIDEO_ID`
- `https://youtu.be/VIDEO_ID`
- `https://www.youtube.com/embed/VIDEO_ID`
- `https://www.youtube.com/shorts/VIDEO_ID`

Always derive the video ID server-side from the submitted URL — never accept a bare video ID from the client.

---

## Friend Key Generation

- Format: `adjective-noun-number` (e.g. `happy-tiger-42`)
- Number range: 1–99
- Use a fixed wordlist for adjectives and nouns (same approach as Songify)
- Keys must be unique per active session — check for collisions before saving
- Keys are case-insensitive on lookup

---

## Logging

Use Go's `slog` package throughout:

```go
slog.Info("playback command broadcast",
    "sessionId", sessionID,
    "command", command,
    "videoId", videoID,
    "viewerCount", count,
)

slog.Error("failed to advance queue",
    "sessionId", sessionID,
    "err", fmt.Errorf("advancing queue: %w", err),
)
```

- Always include `sessionId` in log entries where relevant
- Wrap errors with context: `fmt.Errorf("operation description: %w", err)`
- Never log JWT secrets, passwords, or full tokens — log claims only (session ID, role)

---

## Database & sqlc

- All queries defined in `.sql` files; run `sqlc generate` to regenerate `internal/db/`
- Use DB transactions for multi-table operations (e.g. reordering updates all `position` values)
- Run migrations on startup via the runner in `internal/database/`
- Enable SQLite WAL mode for better read concurrency

---

## Testing

- Unit tests for all service logic: friend key generation, YouTube URL parsing, auto-advance debounce, JWT issuance/validation
- Integration tests for key handler flows using an in-memory SQLite DB
- Test SSE broker: subscription, unsubscription, event delivery, slow consumer handling
- CI runs `go test ./... -v` on every PR
