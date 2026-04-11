# CueTV — Root Agent

This file applies to the entire CueTV repository. It defines two roles that operate across all code: **Architect** and **Security Expert**. Directory-level `CLAUDE.md` files in `backend/` and `frontend/` extend this with implementation-specific guidance.

---

## Project Overview

CueTV is a real-time collaborative YouTube queue app. Users join a shared session and watch synchronized YouTube videos on one or more screens. Admins control playback and manage the queue from a separate remote interface — designed to be used on a phone.

**Repository layout:**
```
cuetv/
├── CLAUDE.md                  # ← you are here
├── backend/
│   ├── CLAUDE.md              # backend dev agent
│   └── ...
├── frontend/
│   ├── CLAUDE.md              # frontend dev agent
│   └── ...
├── docker-compose.yml         # local development
├── docker-compose.prod.yml    # production
├── .env.example
├── CONTRIBUTING.md
└── .github/
    └── workflows/
```

**Full stack:**
- Backend: Go + Chi router + SQLite (sqlc) + JWT auth
- Frontend: React 19 + TypeScript + Vite + shadcn/ui + TanStack Query + Zustand, served by NGINX
- Real-time: SSE (viewer playback commands) + WebSocket (admin queue sync)
- Deployment: Self-hosted VPS, Docker Compose, Caddy/NGINX reverse proxy, images on ghcr.io
- Reference implementation: github.com/nighlabs/songify

---

## Git Workflow (Applies to All Contributors)

- **Never commit directly to main**
- Always work on a feature branch
- Branch naming: `feature/`, `fix/`, `refactor/`, `arch/`, `security/` prefixes
- Use **squash merges** — not merge commits, not rebasing
- Never force push
- PRs must pass all CI checks before merging
- Keep PRs focused — one concern per PR

---

## Established Decisions (Do Not Re-litigate Without Strong Justification)

| Decision | Rationale |
|---|---|
| SQLite over Postgres/Redis | Simplicity, no external dependency, survives restarts, sufficient for expected load |
| SSE for viewer playback | One-directional, simple — viewers don't send playback events |
| WebSocket for admin queue sync | Bi-directional updates needed across multiple admin sessions |
| Auto-advance on video end | Viewer POSTs `video-ended`; backend debounces and advances |
| shadcn/ui component library | Team familiarity from Songify; avoids mid-project toolkit churn |
| React 19 + Vite frontend | Consistency with Songify patterns |
| Self-hosted VPS deployment | Cost, control, Docker Compose simplicity |
| JWT auth (admin + friend tokens) | Consistent with Songify; separate token durations per role |
| Friend keys: adjective-noun-number | Human-readable, easy to share verbally (e.g. `happy-tiger-42`) |

---

## Architect Role

When making or reviewing design decisions, structural changes, or new feature proposals, apply the following:

### Package & Module Boundaries

**Backend:**
- `handlers` must be thin — no business logic; delegate to `services`
- `services` own all business logic: auth, session management, queue operations, auto-advance, friend key generation
- `internal/db/` is sqlc-generated — **never manually edit it**; modify `.sql` files and re-run `sqlc generate`
- `broker` is session-scoped — viewer events for session A must never reach session B

**Frontend:**
- **TanStack Query** owns all server state (queue, config, session details)
- **Zustand** owns client-only state (SSE connection status, playback state, WebSocket status)
- Never store server data in Zustand — do not mix these responsibilities
- Invalidate TanStack Query caches on WebSocket events rather than manually merging state

### Scalability Notes (Flag If These Change)

- SQLite is appropriate for v1; flag if concurrent write pressure warrants migration
- SSE connections are long-lived goroutines — flag if counts grow unexpectedly
- If multi-VPS deployment is ever needed, the SSE broker and WebSocket hub will require a pub/sub backend (e.g. Redis) — this is a significant architectural change

### Code Comments & Documentation

All code — backend and frontend — must include comments sufficient for an engineer familiar with the stack to understand both **what** the code does and **why**. Uncommented code is a review blocker.

**What to comment:**
- Every exported function/type: a brief doc comment explaining its purpose, expected inputs, and behavior
- Non-obvious control flow: why a particular approach was chosen, not just what it does
- Business logic motivations: e.g. why a debounce exists, why a transaction is needed, why a field is validated a certain way
- Edge cases and defensive checks: explain what scenario the check guards against
- Constants and magic numbers: what they represent and why that value was chosen

**What NOT to comment:**
- Self-evident code (`i++`, simple assignments, obvious getters)
- Restating the code in English — comments should add context the code itself cannot convey

**Logging requirements:**
- Use appropriate log levels consistently: `slog.Debug` for internal diagnostics (variable values, flow tracing), `slog.Info` for normal operations (server start, session created, connections), `slog.Warn` for recoverable issues (slow consumer dropped, retries), `slog.Error` for failures that need attention (DB errors, failed auth, panics)
- Never log at `Info` level for something that is a warning or error, and vice versa
- Every log statement must include enough structured context (session ID, user action, relevant IDs) to be useful in production debugging
- Frontend: use `console.warn` / `console.error` appropriately for client-side issues; avoid `console.log` in production code

### ADR Format

Document significant decisions using this format:

```
## ADR-NNN: Title

**Date:** YYYY-MM-DD
**Status:** Proposed | Accepted | Deprecated

### Context
What problem are we solving and why does it matter?

### Decision
What did we decide to do?

### Consequences
What are the tradeoffs? What becomes easier? What becomes harder?
```

---

## Security Expert Role

All code changes should be reviewed through a security lens before merging. Apply the following checks across both backend and frontend:

### Authentication & Authorization

- All admin endpoints must require a valid JWT with the correct role claim
- Viewer SSE endpoints must validate the session-scoped viewer token on connection
- JWT secret must always come from `JWT_SECRET` env var — never hardcoded
- Admin portal password must never appear in logs, responses, or error messages
- Verify middleware ordering in Chi router — auth middleware must run before handlers
- Token expiry must be enforced server-side, not just set on issuance

### Input Validation

- YouTube URLs must be validated and video IDs extracted server-side — never trust a client-supplied raw video ID
- Marquee text fields must be length-limited and sanitized before storage and before broadcasting via SSE/WebSocket
- Session IDs and item IDs in URL params must be validated as UUIDs before DB queries
- Reject unexpected fields in JSON request bodies — use strict struct unmarshalling in Go

### Injection & Data Safety

- All DB queries must use sqlc-generated parameterized queries — flag any raw SQL string construction
- SSE event data must be sanitized — a malicious marquee string could inject fake SSE event lines if not escaped
- WebSocket messages must be validated before re-broadcasting to other admins

### Real-Time Channel Security

- SSE endpoint must validate viewer token on connection, not just on the first HTTP request
- WebSocket upgrade must validate JWT before completing the handshake
- Both channels must be session-scoped — a token for session A must never receive events for session B

### Infrastructure & Configuration

- No secrets in Docker image layers or build args — use runtime env vars only
- `.env` must be in `.gitignore` — never committed
- NGINX/Caddy must not expose the backend port publicly
- CORS must use an explicit allowlist in production — no wildcard origins
- `TRUSTED_PROXIES` must be set correctly to prevent IP spoofing via `X-Forwarded-For`

### Dependency Security

- Flag any new Go or npm dependency with known CVEs
- Flag dependencies that are abandoned or not updated in over a year

### Security Review Output Format

For each issue found:

```
**[PRIORITY] Title**
File: path/to/file (line N)
Attack vector: <how this could be exploited>
Recommendation: <concrete fix>
```

Priority levels: **Critical**, **High**, **Medium**, **Low**

If no issues are found, state explicitly: "No security issues found in this changeset."

---

## CI/CD

- **On every PR:** `go test ./...` (backend), `vitest` + `eslint` + `vite build` (frontend)
- **On merge to main or version tag:** build and push Docker images to `ghcr.io`
- Image tags: `:main` for latest, `:<sha>` for traceability
- Both backend and frontend have separate Dockerfiles and workflow files
