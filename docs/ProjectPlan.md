# CueTV
## Collaborative YouTube Queue — Project Plan & Requirements

*Version 0.1 · April 2026*

---

## 1. Project Overview

CueTV is a real-time collaborative YouTube queue application designed for communal viewing experiences. Users in a shared "room" can watch synchronized YouTube videos on one or more screens while admins control playback and manage the queue from a separate remote interface. The experience is designed to be both local (same room, multiple screens) and distributed (each participant in their own living room, all watching together).

### 1.1 Core Concept

- A viewer page displays a YouTube video player in cinematic/fullscreen mode
- Marquee overlays at the top and/or bottom display metadata about the current or upcoming video
- An admin remote interface controls playback and queue management
- Multiple viewer screens can follow the same queue simultaneously
- Queue changes propagate in real time to all connected admin sessions

### 1.2 Inspiration & Reference

The project structure, logging patterns, containerization strategy, and CI/CD pipelines will be modeled after the Songify project (github.com/nighlabs/songify). Key patterns to carry over:

- Go backend with Chi router, organized into `cmd/server` entry point and `internal/` packages (`broker`, `config`, `database`, `handlers`, `middleware`, `models`, `router`, `services`)
- SQLite via sqlc for type-safe generated queries
- JWT-based authentication with separate admin and friend token durations, configured via environment variables
- Friend join keys using a memorable adjective-noun-number format (e.g. `happy-tiger-42`)
- SSE event broker pattern for real-time updates to connected clients
- React 19 + TypeScript frontend built with Vite, TanStack Query for server state, Zustand for client state, shadcn/ui components
- Docker Compose with backend and frontend containers; backend mounts a named volume for SQLite persistence; frontend depends on backend health check
- Separate `docker-compose.yml` (local dev) and `docker-compose.prod.yml` (production)
- GitHub Actions CI: runs Go tests, frontend vitest + eslint + build on every PR; publishes Docker images to GitHub Container Registry on merge to main or version tags
- Git workflow: feature branches, squash merges, never force push, never commit directly to main

---

## 2. Feature Requirements

### 2.1 Viewer Page

#### 2.1.1 YouTube Player

- Embed the YouTube IFrame Player API
- Display in cinematic/expanded mode filling the full width of its container
- Require a single manual user click to initiate playback (browser autoplay policy compliance)
- After initial play, all subsequent video transitions are controlled programmatically via SSE commands — no additional user interaction required
- Support the following player commands triggered remotely: Play, Pause, Load next video ID, Load previous video ID

#### 2.1.2 Marquee Overlays

- Top marquee bar: optional, toggled per room configuration
- Bottom marquee bar: optional, toggled per room configuration
- Each marquee can independently be configured to pull text from either the current video or the next video in the queue
- Each marquee supports a short label/tag displayed to the left of the scrolling text (e.g. "Now Playing" or "Up Next")
- Marquee text scrolls horizontally if content overflows
- Overlay configuration is set globally per room, not per video

#### 2.1.3 Multi-Screen Support

- Multiple viewer browser tabs or devices can connect to the same room queue
- All viewers receive the same SSE playback commands simultaneously
- Viewers may start slightly out of sync due to independent play initiation, but will re-sync on each "next video" command
- No attempt is made to force millisecond-level sync — natural sync-on-transition is the intended behavior

### 2.2 Admin Remote Interface

#### 2.2.1 Playback Controls

- Play button
- Pause button
- Skip to next video in queue
- Rewind to previous video in queue

#### 2.2.2 Queue Management

- Add videos to the queue by pasting a YouTube URL
- Backend extracts the YouTube video ID from the pasted URL
- Each queued video has an associated text field for marquee display
- Drag-and-drop reordering of queued videos
- Insert a video at any position in the queue (not just at the end)
- The currently playing video continues uninterrupted during queue reordering
- Queue state updates propagate in real time to all connected admin interfaces via WebSocket

#### 2.2.3 Room Configuration

- Toggle top marquee on or off
- Toggle bottom marquee on or off
- Set the label/tag shown to the left of each marquee
- Configure each marquee to source text from the current or next video

### 2.3 Authentication & Session Model

Authentication follows the same JWT + session pattern used in Songify:

- An admin authenticates via an admin portal password (set via environment variable), then creates a new session
- Session creation returns a JWT admin token (configurable duration, default 7 days) and a memorable friend join key in adjective-noun-number format (e.g. `happy-tiger-42`)
- Co-admins join using the friend key via `POST /api/sessions/join`, receiving their own JWT
- Each session generates a viewer URL containing the session ID — shareable with anyone who should watch, no login required
- The viewer URL grants SSE subscription access only; no queue editing or playback control
- JWT secret and admin portal password are configured via environment variables (`JWT_SECRET`, `ADMIN_PORTAL_PASSWORD`)
- Admin token duration and friend token duration are independently configurable via environment variables

### 2.4 Future Considerations (Out of Scope for v1)

- AI-powered YouTube search: allow users to type a search query and receive video suggestions rather than requiring a direct URL paste

---

## 3. Technical Architecture

### 3.1 System Diagram (Conceptual)

| Layer | Technology | Responsibility |
|---|---|---|
| Frontend Viewer | React 19 + TypeScript (Vite) + shadcn/ui served via NGINX | YouTube player, marquee overlays, SSE client for playback commands |
| Frontend Admin | React 19 + TypeScript (Vite) + shadcn/ui served via NGINX | Queue UI, drag-and-drop, playback controls, WebSocket client for queue sync |
| Backend API | Go with Chi router | REST API, SSE broker, WebSocket hub, queue state, business logic, JWT auth |
| Data Store | SQLite via sqlc (type-safe generated queries) | Session state, queue, room config, video metadata |

### 3.2 Backend — Go Service

#### 3.2.1 Responsibilities & Package Structure

Following Songify's `internal/` package layout:

- `cmd/server` — entry point
- `internal/broker` — SSE event broker for fanning out playback commands to all connected viewers
- `internal/config` — environment variable loading and validation
- `internal/database` — SQLite connection and migration management
- `internal/db` — sqlc-generated type-safe query code
- `internal/handlers` — HTTP handlers for all routes
- `internal/middleware` — JWT auth, CORS, rate limiting
- `internal/models` — request/response DTOs
- `internal/router` — route definitions using Chi
- `internal/services` — auth (JWT), friend key generation, session management

#### 3.2.2 Key API Endpoints

| Method | Path | Description |
|---|---|---|
| POST | `/api/sessions` | Create a new session — returns session ID, friend join key, and viewer URL |
| POST | `/api/sessions/join` | Join an existing session as admin using a friend join key |
| POST | `/api/sessions/rejoin` | Rejoin an existing session as admin using JWT |
| GET | `/api/sessions/:id` | Get session details |
| GET | `/api/sessions/:id/events` | SSE stream — viewer subscribes for playback commands |
| GET | `/api/sessions/:id/queue-events` | WebSocket — admin subscribes for live queue updates |
| GET | `/api/sessions/:id/queue` | Fetch current queue state |
| POST | `/api/sessions/:id/queue` | Add a video to the queue |
| PATCH | `/api/sessions/:id/queue` | Reorder queue (submit full new order) |
| DELETE | `/api/sessions/:id/queue/:videoId` | Remove a video from the queue |
| POST | `/api/sessions/:id/playback` | Send playback command (play/pause/next/prev) |
| POST | `/api/sessions/:id/video-ended` | Signal from viewer that current video ended — triggers auto-advance |
| GET | `/api/sessions/:id/config` | Get room configuration |
| PATCH | `/api/sessions/:id/config` | Update room configuration (marquee settings) |
| GET | `/api/health` | Health check |

#### 3.2.3 Real-Time Channels

| Channel | Protocol | Consumers | Events |
|---|---|---|---|
| Playback channel | Server-Sent Events (SSE) | All viewer browser tabs | `play`, `pause`, `load:{videoId}` |
| Queue sync channel | WebSocket | All connected admin UIs | `queue:updated`, `config:updated` |

#### 3.2.4 Logging & Observability

- Use Go's slog package for structured logging, following Songify's error wrapping conventions
- Log all playback command broadcasts with session ID, command type, and video ID
- Log queue mutations with before/after state summaries
- Log SSE and WebSocket connection open/close events

### 3.3 Frontend — React + TypeScript (NGINX)

Following Songify's frontend stack: React 19 + TypeScript, built with Vite, TanStack Query for server state, Zustand for client state, shadcn/ui for UI components. shadcn/ui was explicitly chosen as the component library to avoid the pitfall of picking an unfamiliar or ill-fitting toolkit — the team has prior experience with it from Songify. Output is compiled static assets served by NGINX.

#### 3.3.1 Frontend Package Structure

- `src/components` — shared UI components (queue item, marquee bar, playback controls, video input)
- `src/pages` — viewer page, admin page
- `src/services` — API client, auth helpers
- `src/stores` — Zustand stores for queue state, playback state, session state
- `src/types` — TypeScript type definitions

#### 3.3.2 Viewer Page

- On load: connect to backend SSE endpoint for the session
- Render YouTube IFrame player in cinematic container (100% width, tall fixed height or full viewport)
- Display top and/or bottom marquee bars based on session config fetched on load
- On SSE event `play`: call `player.playVideo()`
- On SSE event `pause`: call `player.pauseVideo()`
- On SSE event `load:{videoId}`: call `player.loadVideoById(videoId)`
- On video end: POST to `/api/sessions/:id/video-ended`; backend auto-advances queue and broadcasts next load command to all viewers

#### 3.3.3 Admin Page

- On load: fetch queue state and session config; connect to WebSocket for live updates
- Render queue as a drag-and-drop ordered list (shadcn/ui + dnd-kit)
- YouTube URL input field — on submit, POST to backend; backend extracts video ID
- Playback control buttons wired to `POST /api/sessions/:id/playback`
- Session config toggles (top marquee, bottom marquee, labels, source video) wired to `PATCH /api/sessions/:id/config`

### 3.4 Containerization

Following Songify's Docker setup exactly:

| Container | Build | Runtime |
|---|---|---|
| backend | golang:alpine multi-stage build | Compiled Go binary; mounts a named Docker volume at `/data` for SQLite persistence |
| frontend | node:alpine Vite build | nginx:alpine serving compiled static assets; proxies `/api` to backend |

- `docker-compose.yml` for local development
- `docker-compose.prod.yml` for production (mirrors Songify pattern)
- Backend health check via wget on `/api/health`; frontend waits for backend healthy before starting
- All configuration via environment variables; `.env.example` documents all required and optional vars
- **Required env vars:** `JWT_SECRET`, `ADMIN_PORTAL_PASSWORD`
- **Optional env vars:** `ADMIN_TOKEN_DURATION` (default 168h), `FRIEND_TOKEN_DURATION` (default 12h), `PORT` (default 3000), `DATABASE_PATH` (default `/data/cuetv.db`), `SENTRY_DSN`, `TRUSTED_PROXIES`

### 3.5 CI/CD — GitHub Actions

Mirroring Songify's GitHub Actions setup:

- On every PR: run `go test ./...` for backend, vitest + eslint + vite build for frontend
- On merge to main or version tag: build and push both Docker images to GitHub Container Registry (`ghcr.io`)
- Image tags: `:main` for latest main branch build, `:<sha>` for traceability
- Separate workflow files for PR checks vs Docker publish
- Git workflow matches Songify's `CLAUDE.md`: feature branches only, squash merges, never commit directly to main, never force push

### 3.6 Self-Hosted Deployment

Production deployment targets a self-hosted VPS (e.g. DigitalOcean, Hetzner) using `docker-compose.prod.yml`, mirroring Songify's production setup.

- A reverse proxy (Caddy recommended for automatic TLS, or NGINX) sits in front of the Docker stack, handling HTTPS termination and routing the domain to the frontend container
- The frontend NGINX container proxies `/api/*` requests to the backend container — no backend port is exposed publicly
- SQLite data persists in a named Docker volume; regular volume backups recommended
- Environment variables are managed via a `.env` file on the host (never committed to git)
- Deployment workflow: `ssh` to VPS → `git pull` → `docker compose -f docker-compose.prod.yml up -d --build`
- GitHub Container Registry images (`ghcr.io`) can be pulled directly in production as an alternative to building on the VPS

---

## 4. Data Model

All state is persisted to a SQLite database, following the same approach used in Songify. This ensures queue and session state survive backend restarts without requiring an external database service.

### 4.1 Session

| Field | Type | Description |
|---|---|---|
| id | string (UUID) | Unique session identifier |
| friendJoinKey | string | Memorable adjective-noun-number key for admins to join (e.g. `happy-tiger-42`) |
| viewerToken | string | Token embedded in viewer URL for read-only SSE access |
| createdAt | timestamp | When the session was created |

### 4.2 Room Config

| Field | Type | Description |
|---|---|---|
| sessionId | string (UUID) | Foreign key to session |
| topMarqueeEnabled | bool | Whether top marquee is shown |
| bottomMarqueeEnabled | bool | Whether bottom marquee is shown |
| topMarqueeLabel | string | Label shown to left of top marquee |
| bottomMarqueeLabel | string | Label shown to left of bottom marquee |
| topMarqueeSource | enum: `current` \| `next` | Which video's text the top marquee displays |
| bottomMarqueeSource | enum: `current` \| `next` | Which video's text the bottom marquee displays |
| currentIndex | int | Index of currently playing video in queue |

### 4.3 QueueItem

| Field | Type | Description |
|---|---|---|
| id | string (UUID) | Unique item identifier |
| sessionId | string (UUID) | Foreign key to session |
| youtubeVideoId | string | Extracted YouTube video ID |
| youtubeUrl | string | Original pasted URL |
| marqueeText | string | Text to show in marquee when this video is current or next |
| position | int | Order position in queue |
| addedAt | timestamp | When the item was added |

---

## 5. Implementation Plan

### Phase 1 — Foundation

1. Set up repository structure mirroring Songify: `backend/cmd/server`, `backend/internal/*`, `frontend/src/*`, docker-compose files, `.env.example`, `CLAUDE.md`, `CONTRIBUTING.md`
2. Scaffold Go backend with Chi router and slog structured logging
3. Set up SQLite with sqlc for type-safe query generation; implement DB migrations
4. Implement session creation, JWT issuance, friend key generation (adjective-noun-number format), and viewer URL generation
5. Implement SSE broker for fanning out playback commands to all connected viewers
6. Implement WebSocket hub for real-time queue sync to all connected admins
7. REST endpoints for queue CRUD, room config, and playback commands
8. YouTube URL parser utility (handle all `youtube.com` and `youtu.be` URL variants)
9. `POST /api/sessions/:id/video-ended` endpoint with debounce logic for auto-advance

### Phase 2 — Frontend

1. Set up React 19 + TypeScript Vite project; configure TanStack Query, Zustand, shadcn/ui
2. Build admin auth flow: admin portal password entry, session creation, friend key display, rejoin flow
3. Build viewer page: YouTube IFrame API integration, cinematic layout, SSE client, auto-advance signal on video end
4. Build marquee overlay components with configurable source (current/next) and label
5. Build admin queue page: queue list with drag-and-drop reorder, add video form (paste URL), per-video marquee text input
6. Wire admin playback controls (play, pause, next, prev) to backend
7. Wire room config toggles (marquee on/off, labels, source) to backend

### Phase 3 — Containerization & CI/CD

1. Write Dockerfile for Go backend (multi-stage golang:alpine → alpine runtime)
2. Write Dockerfile for frontend (node:alpine Vite build → nginx:alpine)
3. Write `docker-compose.yml` for local dev and `docker-compose.prod.yml` for production
4. Set up GitHub Actions: PR checks workflow (go test, vitest, eslint, build)
5. Set up GitHub Actions: Docker image publish workflow (build + push to ghcr.io on main/tag)

### Phase 4 — Polish & Testing

1. End-to-end testing of multi-viewer sync behavior
2. Test queue reordering mid-playback
3. Test multi-admin queue sync
4. Responsive layout testing on mobile admin interface (primary use case)
5. Performance testing of SSE with 10+ concurrent viewer connections
6. Test auto-advance debounce with multiple simultaneous viewer tabs

---

## 6. Decisions Log

| # | Question | Decision |
|---|---|---|
| 1 | Auto-advance on video end? | **Resolved:** Yes. Viewer signals video end to backend via POST; backend auto-advances queue and broadcasts next load command. Debounce required when multiple viewer tabs signal simultaneously. |
| 2 | Persistent storage? | **Resolved:** SQLite, following Songify's approach. Lightweight, no external dependency, survives restarts. |
| 3 | Authentication? | **Resolved:** Session-based JWT model from Songify. Admin creates session, gets adjective-noun-number friend key for co-admins, plus a viewer URL for watchers. |
| 4 | Frontend framework & UI library? | **Resolved:** React 19 + TypeScript + Vite + shadcn/ui — matching Songify. Explicit decision to avoid the pitfall of picking an unfamiliar toolkit mid-project. |
| 5 | Deployment target? | **Resolved:** Self-hosted VPS. Use `docker-compose.prod.yml` with Caddy or NGINX reverse proxy for TLS termination. |
| 6 | Auto-advance debounce? | **Resolved:** Backend must debounce `video-ended` signals to avoid advancing twice when multiple viewer tabs report end simultaneously. |

---

## 7. Non-Functional Requirements

| Requirement | Target |
|---|---|
| SSE latency (playback command to viewer) | < 500ms under normal network conditions |
| Queue sync latency (admin action to all admins) | < 300ms |
| Concurrent viewers per session | 10+ without degradation (v1 target) |
| Browser support | Modern Chromium-based browsers and Firefox (latest 2 versions) |
| Mobile admin usability | Admin interface usable on a phone screen (primary use case) |
| Container startup time | < 5 seconds for both containers |

---

*CueTV — Draft — Subject to Change*
