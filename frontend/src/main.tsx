import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AdminPage } from "@/pages/AdminPage";
import { ViewerPage } from "@/pages/ViewerPage";
import "./index.css";

/**
 * Shared TanStack Query client with defaults tuned for a live session app.
 *
 * - `refetchOnWindowFocus: false` — Prevents automatic refetches when the user
 *   switches tabs or windows. During a live session this would cause disruptive
 *   UI flashes and unnecessary network traffic every time an admin checks
 *   another app on their phone.
 * - `retry: 1` — Limits failed queries to a single retry. The default of 3
 *   adds noticeable delay and console noise, especially on SSE/WebSocket
 *   endpoints that have their own reconnection logic.
 */
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

/**
 * Application routing layout:
 *
 * - `/admin` — The remote-control interface (designed for phone use). Handles
 *   auth, queue management, playback controls, and room configuration.
 * - `/viewer/:sessionId` — The display page shown on the TV/screen. Connects
 *   via SSE to receive playback commands and renders the YouTube player.
 * - `*` (catch-all) — Redirects to `/admin`. The admin page is the primary
 *   entry point; viewer URLs are generated and shared from within a session.
 */
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/admin" element={<AdminPage />} />
          <Route path="/viewer/:sessionId" element={<ViewerPage />} />
          <Route path="*" element={<Navigate to="/admin" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>
);
