# CueTV Frontend Developer

You are the frontend developer for CueTV. This file extends the root `CLAUDE.md` — read that first for project overview, architectural rules, git workflow, and security requirements. This file covers everything specific to the `frontend/` directory.

---

## Stack

- **Framework:** React 19 + TypeScript, built with Vite
- **UI components:** shadcn/ui + Tailwind CSS
- **Server state:** TanStack Query
- **Client state:** Zustand
- **Served by:** NGINX (compiled static assets); `/api/*` proxied to Go backend

---

## Directory Structure

```
frontend/
└── src/
    ├── components/    # Shared UI components
    ├── pages/         # ViewerPage, AdminPage
    ├── services/      # API client, auth helpers, SSE client, WebSocket client
    ├── stores/        # Zustand stores (queue, playback, session)
    ├── types/         # TypeScript type definitions
    └── main.tsx
```

---

## State Management Rules

- **TanStack Query** owns all server state: queue list, session config, session details
- **Zustand** owns client-only state: SSE connection status, WebSocket connection status, current playback state, local UI state
- Never store server data in Zustand — always use TanStack Query for anything from the API
- On WebSocket `queue:updated` event: invalidate TanStack Query `sessions/:id/queue` cache
- On WebSocket `config:updated` event: invalidate TanStack Query `sessions/:id/config` cache

---

## Component Guidelines

- Use **shadcn/ui** primitives for all UI — do not install additional component libraries
- All components must be typed with explicit props interfaces — no `any`
- Prefer small, focused components over large monolithic ones
- Use `React.memo` for queue list items that re-render frequently during drag-and-drop

---

## Viewer Page

**Purpose:** Displayed on the screen(s) being watched. Receives SSE commands and drives the YouTube player.

### YouTube IFrame Player

- Embed via the YouTube IFrame Player API (`YT.Player`)
- Cinematic container: 100vw width, 16:9 aspect ratio, dark background
- Show a large centered play button overlay on load — do not autoplay
- After the first user click, all subsequent playback is driven by SSE commands only

### SSE Client

- Connect to `GET /api/sessions/:id/events?token={viewerToken}` on page load
- Reconnect automatically on disconnect with exponential backoff (max 30s delay)
- Handle events:
  - `play` → `player.playVideo()`
  - `pause` → `player.pauseVideo()`
  - `load:{videoId}` → `player.loadVideoById(videoId)` then `player.playVideo()`
  - `queue:ended` → show an end-of-queue overlay

### Auto-Advance Signal

- On YouTube player `onStateChange` with `YT.PlayerState.ENDED`: POST to `/api/sessions/:id/video-ended` with viewer token
- Fire once per video end — do not retry if the request fails (backend debounces)

### Marquee Overlays

- Render top and bottom marquee bars conditionally based on session config
- CSS `@keyframes` marquee animation — no JavaScript scroll loops
- Each bar shows: label on the left (e.g. "Now Playing") + scrolling text on the right
- Text source (current or next video's `marqueeText`) resolved from queue state

---

## Admin Page

**Purpose:** The remote control. Primarily used on a phone.

### Auth Flow

1. Enter admin portal password → `POST /api/admin/verify`
2. On success: create session (`POST /api/sessions`) or rejoin (`POST /api/sessions/rejoin`)
3. Display friend join key prominently with a copy-to-clipboard button
4. Store JWT in memory (Zustand) — not in localStorage

### Queue Management

- Drag-and-drop reorder using `@dnd-kit/core` + `@dnd-kit/sortable`
- Each queue item shows:
  - Thumbnail: `https://img.youtube.com/vi/{videoId}/default.jpg`
  - Marquee text input (editable inline)
  - Drag handle
  - Delete button
- Add video form: single URL input, validate as YouTube URL client-side, submit on Enter or button click
- On submit: `POST /api/sessions/:id/queue` with `{ url, marqueeText }`
- On reorder: `PATCH /api/sessions/:id/queue` with `{ order: string[] }` (full ordered array of item IDs)

### Playback Controls

- Play, Pause, Previous, Next buttons
- Minimum touch target: 44×44px on all interactive elements
- Wire to `POST /api/sessions/:id/playback` with `{ command: "play"|"pause"|"next"|"prev" }`

### Room Config Panel

- Toggle top/bottom marquee (shadcn/ui `Switch`)
- Label text inputs for each marquee
- Source selector — current or next video (shadcn/ui `Select`)
- Wire to `PATCH /api/sessions/:id/config`

### WebSocket Client

- Connect to `GET /api/sessions/:id/queue-events` with JWT in `Authorization` header
- Reconnect automatically on disconnect with exponential backoff (max 30s delay)
- On `queue:updated`: invalidate TanStack Query queue cache
- On `config:updated`: invalidate TanStack Query config cache

---

## API Reference

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/admin/verify` | None | Verify admin portal password |
| POST | `/api/sessions` | Admin password | Create session → JWT, friend key, viewer URL |
| POST | `/api/sessions/join` | None | Join with friend key → JWT |
| POST | `/api/sessions/rejoin` | JWT | Rejoin existing session |
| GET | `/api/sessions/:id` | JWT | Session details |
| GET | `/api/sessions/:id/events` | Viewer token (query param) | SSE stream |
| GET | `/api/sessions/:id/queue-events` | JWT (header) | WebSocket |
| GET | `/api/sessions/:id/queue` | JWT | Queue state |
| POST | `/api/sessions/:id/queue` | JWT | Add video `{ url, marqueeText }` |
| PATCH | `/api/sessions/:id/queue` | JWT | Reorder `{ order: string[] }` |
| DELETE | `/api/sessions/:id/queue/:itemId` | JWT | Remove item |
| POST | `/api/sessions/:id/playback` | JWT | `{ command: "play"\|"pause"\|"next"\|"prev" }` |
| POST | `/api/sessions/:id/video-ended` | Viewer token | Signal video end |
| GET | `/api/sessions/:id/config` | JWT | Room config |
| PATCH | `/api/sessions/:id/config` | JWT | Update config |

---

## TypeScript Conventions

- Strict mode enabled — no `any`, no `@ts-ignore` without an explanatory comment
- All API response shapes defined in `src/types/` — do not inline types in components
- Use `zod` for runtime validation of critical API responses (e.g. session creation)

---

## Styling

- Tailwind CSS utility classes only — no custom CSS files except for marquee animation keyframes
- Dark theme by default — the viewer page is designed for a dark room
- Mobile-first breakpoints — admin interface must be fully usable at 390px width (iPhone viewport)
- Minimum touch target: 44×44px for all interactive elements on the admin page

---

## Testing

- Unit tests with **vitest** + **React Testing Library**
- Test Zustand store logic
- Test SSE and WebSocket client reconnection logic
- Test YouTube URL validation utility
- CI runs `npm test`, `eslint`, and `vite build` on every PR
