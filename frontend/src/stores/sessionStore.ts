import { create } from "zustand";
import { setAuthToken } from "@/services/api";

/**
 * Holds authentication credentials (JWT, session ID, friend key, viewer
 * token) in memory. Intentionally not persisted to localStorage to
 * prevent XSS-based token theft.
 */
interface SessionState {
  token: string | null;
  sessionId: string | null;
  friendKey: string | null;
  viewerToken: string | null;
  isAuthenticated: boolean;
  setSession: (params: {
    token: string;
    sessionId: string;
    friendKey: string;
    viewerToken: string;
  }) => void;
  clearSession: () => void;
}

/**
 * Zustand store for session auth state. The JWT is stored in memory only
 * (not localStorage) to prevent XSS token theft. On every change the
 * token is synced to the api.ts module-level variable via
 * {@link setAuthToken} so outgoing requests carry the correct header.
 */
export const useSessionStore = create<SessionState>((set) => ({
  token: null,
  sessionId: null,
  friendKey: null,
  viewerToken: null,
  isAuthenticated: false,
  /** Called after successful auth (create or join) to store credentials and sync the API client. */
  setSession: ({ token, sessionId, friendKey, viewerToken }) => {
    setAuthToken(token);
    set({
      token,
      sessionId,
      friendKey,
      viewerToken,
      isAuthenticated: true,
    });
  },
  /** Clears both the Zustand store and the API client's module-level token. */
  clearSession: () => {
    setAuthToken(null);
    set({
      token: null,
      sessionId: null,
      friendKey: null,
      viewerToken: null,
      isAuthenticated: false,
    });
  },
}));
