import { create } from "zustand";

type PlaybackState = "idle" | "playing" | "paused";

interface SocketState {
  wsConnected: boolean;
  setWsConnected: (connected: boolean) => void;
  /** Tracks the admin's last-issued playback command so the UI can
   *  highlight the active control button and show "Now Playing" only
   *  when playback has actually started. */
  playbackState: PlaybackState;
  setPlaybackState: (state: PlaybackState) => void;
}

/**
 * Tracks WebSocket connection status and admin-side playback state.
 * Playback state is derived from the admin's own commands (not from
 * the viewer's actual player state, which we don't have access to).
 */
export const useSocketStore = create<SocketState>((set) => ({
  wsConnected: false,
  setWsConnected: (connected) => set({ wsConnected: connected }),
  playbackState: "idle",
  setPlaybackState: (playbackState) => set({ playbackState }),
}));
