import type {
  AdminVerifyResponse,
  CreateSessionResponse,
  JoinSessionResponse,
  PlaybackCommandRequest,
  QueueItem,
  RejoinSessionResponse,
  RoomConfig,
  SessionResponse,
  UpdateRoomConfigRequest,
} from "@/types/api";

/**
 * Module-level shadow of the Zustand session store's JWT. Kept in sync
 * via {@link setAuthToken} so the API client can attach the token to
 * outgoing requests without importing the store (avoids circular deps).
 */
let authToken: string | null = null;

/**
 * Called by the session store on login/logout to keep the API client's
 * module-level token in sync with the Zustand store.
 */
export function setAuthToken(token: string | null) {
  authToken = token;
}

/**
 * Returns the current JWT held by the API client. Useful for passing
 * the token to WebSocket or SSE connections that need it out-of-band.
 */
export function getAuthToken(): string | null {
  return authToken;
}

/**
 * Generic fetch wrapper that injects the JWT Authorization header (when
 * available), throws on non-OK responses with the response body as the
 * error message, and returns `undefined` for 204 No Content responses.
 */
async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  if (authToken) {
    headers["Authorization"] = `Bearer ${authToken}`;
  }

  const response = await fetch(path, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const text = await response.text();
    console.error(`API error: ${response.status} ${response.statusText}`, { path, text });
    throw new Error(text || `HTTP ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

/** Verify the admin portal password. POST /api/admin/verify */
export function verifyAdmin(password: string) {
  return request<AdminVerifyResponse>("/api/admin/verify", {
    method: "POST",
    body: JSON.stringify({ password }),
  });
}

/** Create a new session with admin credentials. POST /api/sessions */
export function createSession(password: string) {
  return request<CreateSessionResponse>("/api/sessions", {
    method: "POST",
    body: JSON.stringify({ password }),
  });
}

/** Join an existing session using a friend key. POST /api/sessions/join */
export function joinSession(friendKey: string) {
  return request<JoinSessionResponse>("/api/sessions/join", {
    method: "POST",
    body: JSON.stringify({ friendKey }),
  });
}

/** Rejoin a previously created session using the current JWT. POST /api/sessions/rejoin */
export function rejoinSession() {
  return request<RejoinSessionResponse>("/api/sessions/rejoin", {
    method: "POST",
  });
}

/** Fetch session details by ID. GET /api/sessions/:id */
export function getSession(sessionId: string) {
  const sid = encodeURIComponent(sessionId);
  return request<SessionResponse>(`/api/sessions/${sid}`);
}

/** Fetch the full queue for a session. GET /api/sessions/:id/queue
 *  Accepts an optional viewer token for unauthenticated viewer access. */
export function getQueue(sessionId: string, viewerToken?: string) {
  const sid = encodeURIComponent(sessionId);
  const tokenParam = viewerToken ? `?token=${encodeURIComponent(viewerToken)}` : "";
  return request<QueueItem[]>(`/api/sessions/${sid}/queue${tokenParam}`);
}

/** Add a YouTube video to the session queue. POST /api/sessions/:id/queue */
export function addToQueue(
  sessionId: string,
  url: string,
  marqueeText: string
) {
  const sid = encodeURIComponent(sessionId);
  return request<QueueItem>(`/api/sessions/${sid}/queue`, {
    method: "POST",
    body: JSON.stringify({ url, marqueeText }),
  });
}

/** Reorder the session queue by providing the full ordered list of item IDs. PATCH /api/sessions/:id/queue */
export function reorderQueue(sessionId: string, order: string[]) {
  const sid = encodeURIComponent(sessionId);
  return request<void>(`/api/sessions/${sid}/queue`, {
    method: "PATCH",
    body: JSON.stringify({ order }),
  });
}

/** Update the marquee text for a single queue item. PATCH /api/sessions/:id/queue/:itemId */
export function updateQueueItemMarqueeText(
  sessionId: string,
  itemId: string,
  marqueeText: string
) {
  const sid = encodeURIComponent(sessionId);
  const iid = encodeURIComponent(itemId);
  return request<void>(`/api/sessions/${sid}/queue/${iid}`, {
    method: "PATCH",
    body: JSON.stringify({ marqueeText }),
  });
}

/** Delete a single item from the session queue. DELETE /api/sessions/:id/queue/:itemId */
export function deleteQueueItem(sessionId: string, itemId: string) {
  const sid = encodeURIComponent(sessionId);
  const iid = encodeURIComponent(itemId);
  return request<void>(`/api/sessions/${sid}/queue/${iid}`, {
    method: "DELETE",
  });
}

/** Send a playback command (play/pause/next/prev) to the session. POST /api/sessions/:id/playback */
export function sendPlaybackCommand(
  sessionId: string,
  command: PlaybackCommandRequest["command"]
) {
  const sid = encodeURIComponent(sessionId);
  return request<void>(`/api/sessions/${sid}/playback`, {
    method: "POST",
    body: JSON.stringify({ command }),
  });
}

/** Signal that the current video has ended on the viewer. POST /api/sessions/:id/video-ended */
export function signalVideoEnded(sessionId: string, viewerToken: string) {
  const sid = encodeURIComponent(sessionId);
  return fetch(
    `/api/sessions/${sid}/video-ended?token=${encodeURIComponent(viewerToken)}`,
    { method: "POST" }
  );
}

/** Fetch the room configuration for a session. GET /api/sessions/:id/config
 *  Accepts an optional viewer token for unauthenticated viewer access. */
export function getRoomConfig(sessionId: string, viewerToken?: string) {
  const sid = encodeURIComponent(sessionId);
  const tokenParam = viewerToken ? `?token=${encodeURIComponent(viewerToken)}` : "";
  return request<RoomConfig>(`/api/sessions/${sid}/config${tokenParam}`);
}

/** Update the room configuration for a session. PATCH /api/sessions/:id/config */
export function updateRoomConfig(
  sessionId: string,
  config: UpdateRoomConfigRequest
) {
  const sid = encodeURIComponent(sessionId);
  return request<RoomConfig>(`/api/sessions/${sid}/config`, {
    method: "PATCH",
    body: JSON.stringify(config),
  });
}
