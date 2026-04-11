import { create } from "zustand";

interface PlaybackState {
  sseConnected: boolean;
  setSseConnected: (connected: boolean) => void;
}

/**
 * Tracks the SSE connection status so the viewer page can show a
 * connected/disconnected UI indicator.
 */
export const usePlaybackStore = create<PlaybackState>((set) => ({
  sseConnected: false,
  setSseConnected: (connected) => set({ sseConnected: connected }),
}));
