import { parseWSEvent, type WSEvent } from "@/types/events";

type WSEventHandler = (event: WSEvent) => void;

/**
 * Manages the WebSocket connection to the admin queue-events endpoint
 * (`/api/sessions/:id/queue-events`) with automatic reconnection on
 * close. Events are parsed via {@link parseWSEvent} and forwarded to
 * the provided callback; unrecognized messages are silently skipped.
 */
export class WebSocketClient {
  private ws: WebSocket | null = null;
  private sessionId: string;
  private token: string;
  private onEvent: WSEventHandler;
  private onStatusChange: (connected: boolean) => void;
  private reconnectAttempts = 0;
  /** Maximum delay between reconnection attempts (30 seconds). */
  private maxReconnectDelay = 30000;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(
    sessionId: string,
    token: string,
    onEvent: WSEventHandler,
    onStatusChange: (connected: boolean) => void
  ) {
    this.sessionId = sessionId;
    this.token = token;
    this.onEvent = onEvent;
    this.onStatusChange = onStatusChange;
  }

  /**
   * Closes any existing connection, detects ws/wss based on the current
   * page protocol, and opens a new WebSocket to the session-scoped
   * queue-events endpoint. Resets the reconnect counter on successful open.
   */
  connect() {
    this.disconnect();

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${protocol}//${window.location.host}/api/sessions/${encodeURIComponent(this.sessionId)}/queue-events?token=${encodeURIComponent(this.token)}`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.onStatusChange(true);
    };

    this.ws.onmessage = (event) => {
      const parsed = parseWSEvent(event.data);
      if (parsed) {
        this.onEvent(parsed);
      }
    };

    this.ws.onclose = () => {
      this.onStatusChange(false);
      this.ws = null;
      this.scheduleReconnect();
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  /**
   * Clears any pending reconnect timer and closes the WebSocket.
   * Notifies the status callback that the connection is closed.
   */
  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
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
    console.warn(`WebSocket disconnected, reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }
}
