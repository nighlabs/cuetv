import { parseSSEEvent, type SSEEvent } from "@/types/events";

type SSEEventHandler = (event: SSEEvent) => void;

/**
 * Manages the SSE connection to the viewer events endpoint
 * (`/api/sessions/:id/events`) with automatic reconnection on failure.
 * Events are parsed via {@link parseSSEEvent} and forwarded to the
 * provided callback; unrecognized events are silently skipped.
 */
export class SSEClient {
  private eventSource: EventSource | null = null;
  private sessionId: string;
  private viewerToken: string;
  private onEvent: SSEEventHandler;
  private onStatusChange: (connected: boolean) => void;
  private reconnectAttempts = 0;
  /** Maximum delay between reconnection attempts (30 seconds). */
  private maxReconnectDelay = 30000;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(
    sessionId: string,
    viewerToken: string,
    onEvent: SSEEventHandler,
    onStatusChange: (connected: boolean) => void
  ) {
    this.sessionId = sessionId;
    this.viewerToken = viewerToken;
    this.onEvent = onEvent;
    this.onStatusChange = onStatusChange;
  }

  /**
   * Closes any existing connection, then opens a new EventSource to the
   * session-scoped SSE endpoint. Resets the reconnect counter on success.
   */
  connect() {
    this.disconnect();

    const url = `/api/sessions/${encodeURIComponent(this.sessionId)}/events?token=${encodeURIComponent(this.viewerToken)}`;
    this.eventSource = new EventSource(url);

    this.eventSource.onmessage = (event) => {
      const parsed = parseSSEEvent(event.data);
      if (parsed) {
        this.onEvent(parsed);
      }
    };

    this.eventSource.onopen = () => {
      this.reconnectAttempts = 0;
      this.onStatusChange(true);
    };

    this.eventSource.onerror = () => {
      this.onStatusChange(false);
      this.eventSource?.close();
      this.eventSource = null;
      this.scheduleReconnect();
    };
  }

  /**
   * Clears any pending reconnect timer and closes the EventSource.
   * Notifies the status callback that the connection is closed.
   */
  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    this.onStatusChange(false);
  }

  /**
   * Schedules a reconnection attempt using exponential backoff
   * (1s, 2s, 4s, 8s, ...) capped at {@link maxReconnectDelay} (30s).
   */
  private scheduleReconnect() {
    const delay = Math.min(
      1000 * Math.pow(2, this.reconnectAttempts),
      this.maxReconnectDelay
    );
    this.reconnectAttempts++;
    console.warn(`SSE disconnected, reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }
}
