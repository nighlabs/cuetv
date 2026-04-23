/**
 * Discriminated union of viewer playback events received via SSE.
 * Each variant maps to a command the backend broadcasts to all viewers
 * in a session: play/pause the player, load a new video, or signal
 * that the queue has ended.
 */
export type SSEEvent =
  | { type: "play" }
  | { type: "pause" }
  | { type: "load"; videoId: string }
  | { type: "queue:ended" }
  | { type: "connected" };

/**
 * Discriminated union of admin sync events received via WebSocket.
 * These notify the admin UI that server-side state has changed so
 * TanStack Query caches can be invalidated.
 */
export type WSEvent =
  | { type: "queue:updated" }
  | { type: "config:updated" };

/**
 * Parses a raw SSE `data` string into a typed {@link SSEEvent}.
 * The backend sends plain-text event payloads (e.g. "play", "load:{videoId}").
 * Returns `null` for unrecognized event formats so the caller can skip them.
 */
export function parseSSEEvent(data: string): SSEEvent | null {
  if (data === "play") return { type: "play" };
  if (data === "pause") return { type: "pause" };
  if (data === "connected") return { type: "connected" };
  if (data === "queue:ended") return { type: "queue:ended" };
  if (data.startsWith("load:")) {
    return { type: "load", videoId: data.slice(5) };
  }
  console.warn(`Unknown SSE event: ${data}`);
  return null;
}

/**
 * Parses a raw WebSocket message (JSON) into a typed {@link WSEvent}.
 * Returns `null` for unrecognized or malformed messages so the caller
 * can safely skip them.
 */
export function parseWSEvent(data: string): WSEvent | null {
  try {
    const parsed: unknown = JSON.parse(data);
    if (typeof parsed !== "object" || parsed === null || !("type" in parsed)) {
      return null;
    }
    const { type } = parsed as { type: unknown };
    if (type === "queue:updated") return { type: "queue:updated" };
    if (type === "config:updated") return { type: "config:updated" };
    return null;
  } catch {
    return null;
  }
}
