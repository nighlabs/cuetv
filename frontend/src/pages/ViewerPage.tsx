import { useEffect, useRef, useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { SSEClient } from "@/services/sseClient";
import { getQueue, getRoomConfig, signalVideoEnded } from "@/services/api";
import { usePlaybackStore } from "@/stores/playbackStore";
import { PlayerOverlay } from "@/components/PlayerOverlay";
import type { SSEEvent } from "@/types/events";
import { Play } from "lucide-react";

/**
 * Augment the global Window interface with YouTube IFrame Player API types.
 * The YT namespace is injected by the YouTube IFrame API script at runtime,
 * so we declare it here to get type-safety in the rest of the file.
 */
declare global {
  interface Window {
    YT: typeof YT;
    onYouTubeIframeAPIReady: () => void;
  }
}

/**
 * ViewerPage — the full-screen viewer display shown on the TV/screen.
 *
 * Connects to the backend via SSE to receive real-time playback commands
 * (play, pause, load, queue:ended) and drives an embedded YouTube IFrame
 * Player accordingly. Renders optional marquee overlays based on room config.
 */
export function ViewerPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  /**
   * Read the viewer token from the URL query string once on mount.
   * useState with an initializer function ensures this only runs once,
   * avoiding re-parsing on every render.
   */
  const [viewerToken] = useState(
    () => new URLSearchParams(window.location.search).get("token") || ""
  );
  const playerRef = useRef<YT.Player | null>(null);
  const playerContainerRef = useRef<HTMLDivElement>(null);
  const sseRef = useRef<SSEClient | null>(null);
  const [ytReady, setYtReady] = useState(() => !!window.YT);
  /**
   * Gates the first user interaction required by browser autoplay policy.
   * The play-button overlay is shown until the user clicks, after which
   * all subsequent playback is driven entirely by SSE commands.
   */
  const [started, setStarted] = useState(false);
  const [queueEnded, setQueueEnded] = useState(false);
  /**
   * Prevents duplicate video-ended signals for the same video.
   * Set to true when onStateChange fires ENDED; reset to false on each
   * new "load" SSE event so the next video can trigger its own signal.
   */
  const videoEndedFiredRef = useRef(false);
  const { sseConnected, setSseConnected } = usePlaybackStore();

  const { data: queue = [] } = useQuery({
    queryKey: ["viewer-queue", sessionId],
    queryFn: () => getQueue(sessionId!),
    enabled: !!sessionId,
  });

  const { data: config } = useQuery({
    queryKey: ["viewer-config", sessionId],
    queryFn: () => getRoomConfig(sessionId!),
    enabled: !!sessionId,
  });

  // Load YouTube IFrame API script if not already present. The callback
  // is an external-system subscription (YT API notifying us it's ready),
  // so setState inside the callback is correct — only the synchronous
  // early-return case was moved to the useState initializer above.
  useEffect(() => {
    if (ytReady) return;

    window.onYouTubeIframeAPIReady = () => setYtReady(true);

    const script = document.createElement("script");
    script.src = "https://www.youtube.com/iframe_api";
    document.head.appendChild(script);
  }, [ytReady]);

  // Initialize player
  useEffect(() => {
    if (!ytReady || !playerContainerRef.current || playerRef.current) return;

    playerRef.current = new window.YT.Player(playerContainerRef.current, {
      width: "100%",
      height: "100%",
      /**
       * autoplay: 0 — we cannot autoplay without a user gesture; the play
       * overlay handles the first interaction, then SSE drives playback.
       * controls: 0 — playback is fully controlled by SSE commands from the
       * admin remote; native YouTube controls would conflict.
       */
      playerVars: {
        autoplay: 0,
        controls: 0,
        modestbranding: 1,
        rel: 0,
        showinfo: 0,
      },
      events: {
        onStateChange: (event: YT.OnStateChangeEvent) => {
          if (
            event.data === window.YT.PlayerState.ENDED &&
            !videoEndedFiredRef.current
          ) {
            videoEndedFiredRef.current = true;
            if (sessionId && viewerToken) {
              signalVideoEnded(sessionId, viewerToken);
            }
          }
        },
      },
    });
  }, [ytReady, sessionId, viewerToken]);

  /**
   * Maps incoming SSE events to YouTube player actions:
   *  - "play"        → resume playback
   *  - "pause"       → pause playback
   *  - "load"        → load a new video by ID and start playing
   *  - "queue:ended" → show the end-of-queue overlay
   */
  const handleSSEEvent = useCallback(
    (event: SSEEvent) => {
      const player = playerRef.current;
      if (!player) return;

      switch (event.type) {
        case "play":
          player.playVideo();
          break;
        case "pause":
          player.pauseVideo();
          break;
        case "load":
          videoEndedFiredRef.current = false;
          setQueueEnded(false);
          player.loadVideoById(event.videoId);
          player.playVideo();
          break;
        case "queue:ended":
          setQueueEnded(true);
          break;
      }
    },
    []
  );

  /**
   * SSE connection lifecycle: connects to the backend event stream on mount
   * (once sessionId and viewerToken are available) and disconnects on cleanup.
   * The SSEClient handles reconnection with exponential backoff internally.
   */
  useEffect(() => {
    if (!sessionId || !viewerToken) return;

    const sse = new SSEClient(
      sessionId,
      viewerToken,
      handleSSEEvent,
      setSseConnected
    );
    sseRef.current = sse;
    sse.connect();

    return () => {
      sse.disconnect();
      sseRef.current = null;
    };
  }, [sessionId, viewerToken, handleSSEEvent, setSseConnected]);

  const handleStart = () => {
    setStarted(true);
    playerRef.current?.playVideo();
  };

  return (
    <div className="relative h-screen w-screen bg-black">
      {/* Player */}
      <div className="flex h-full w-full items-center justify-center">
        <div className="relative w-full" style={{ aspectRatio: "16/9" }}>
          <div ref={playerContainerRef} className="h-full w-full" />

          {/* Marquee overlays */}
          {config && <PlayerOverlay config={config} queue={queue} />}
        </div>
      </div>

      {/* Play button overlay */}
      {!started && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/80">
          <button
            onClick={handleStart}
            className="rounded-full bg-white/10 p-8 transition-colors hover:bg-white/20"
          >
            <Play size={64} className="text-white" />
          </button>
        </div>
      )}

      {/* Queue ended overlay */}
      {queueEnded && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/80">
          <p className="text-2xl text-white">Queue ended</p>
        </div>
      )}

      {/* Connection status */}
      <div className="absolute right-4 top-4 z-20">
        <span
          className={`h-2 w-2 inline-block rounded-full ${sseConnected ? "bg-green-500" : "bg-red-500"}`}
        />
      </div>
    </div>
  );
}
