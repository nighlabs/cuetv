/** Response from POST /api/sessions — returned after creating a new session. */
export interface CreateSessionResponse {
  sessionId: string;
  token: string;
  friendKey: string;
  viewerToken: string;
}

/** Response from POST /api/sessions/join — returned after joining via friend key. */
export interface JoinSessionResponse {
  sessionId: string;
  token: string;
}

/** Response from POST /api/sessions/rejoin — returned when re-authenticating with an existing JWT. */
export interface RejoinSessionResponse {
  sessionId: string;
  token: string;
  friendKey: string;
  viewerToken: string;
}

/** Response from GET /api/sessions/:id — session details. */
export interface SessionResponse {
  id: string;
  friendKey: string;
  viewerToken: string;
  createdAt: string;
}

/** A single item in the session queue, returned by GET /api/sessions/:id/queue and POST /api/sessions/:id/queue. */
export interface QueueItem {
  id: string;
  youtubeVideoId: string;
  youtubeUrl: string;
  marqueeText: string;
  position: number;
  addedAt: string;
}

/** Response from GET /api/sessions/:id/config and PATCH /api/sessions/:id/config — room display configuration. */
export interface RoomConfig {
  topMarqueeEnabled: boolean;
  bottomMarqueeEnabled: boolean;
  topMarqueeLabel: string;
  bottomMarqueeLabel: string;
  topMarqueeSource: "current" | "next";
  bottomMarqueeSource: "current" | "next";
  currentIndex: number;
}

/** Request body for PATCH /api/sessions/:id/config — all fields are optional partial updates. */
export interface UpdateRoomConfigRequest {
  topMarqueeEnabled?: boolean;
  bottomMarqueeEnabled?: boolean;
  topMarqueeLabel?: string;
  bottomMarqueeLabel?: string;
  topMarqueeSource?: "current" | "next";
  bottomMarqueeSource?: "current" | "next";
}

/** Request body for POST /api/sessions/:id/playback — send a playback command to viewers. */
export interface PlaybackCommandRequest {
  command: "play" | "pause" | "next" | "prev";
}

/** Response from POST /api/admin/verify — indicates whether the admin password is correct. */
export interface AdminVerifyResponse {
  valid: boolean;
}
