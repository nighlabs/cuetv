import { create } from "zustand";

interface SocketState {
  wsConnected: boolean;
  setWsConnected: (connected: boolean) => void;
}

/**
 * Tracks the WebSocket connection status so the admin page can show a
 * connected/disconnected UI indicator.
 */
export const useSocketStore = create<SocketState>((set) => ({
  wsConnected: false,
  setWsConnected: (connected) => set({ wsConnected: connected }),
}));
