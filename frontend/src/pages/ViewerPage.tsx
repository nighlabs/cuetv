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
 *
 * The player is NOT created until we have both a video ID to load and a user
 * gesture (click on the play overlay). This avoids YouTube's error state when
 * creating an empty player, and satisfies browser autoplay policy.
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
   * Tracks whether the user has clicked the play overlay, satisfying the
   * browser autoplay policy. The player won't be created until both this
   * is true AND we have a video ID to load.
   */
  const [userClicked, setUserClicked] = useState(false);

  /**
   * The video ID to load into the player. Set by the first SSE "load" event
   * or derived from the current queue position. The player is created with
   * this ID to avoid YouTube's empty-player error.
   */
  const [pendingVideoId, setPendingVideoId] = useState<string | null>(null);

  /** Tracks whether the YT.Player has been created. Using state (not a ref)
   *  so the overlay re-renders when the player is ready. Set via onReady
   *  callback from the YT.Player constructor, not directly in an effect. */
  const [playerCreated, setPlayerCreated] = useState(false);

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
    queryFn: () => getQueue(sessionId!, viewerToken),
    enabled: !!sessionId && !!viewerToken,
  });

  const { data: config } = useQuery({
    queryKey: ["viewer-config", sessionId],
    queryFn: () => getRoomConfig(sessionId!, viewerToken),
    enabled: !!sessionId && !!viewerToken,
  });

  // Load YouTube IFrame API script if not already present.
  useEffect(() => {
    if (ytReady) return;

    window.onYouTubeIframeAPIReady = () => setYtReady(true);

    const script = document.createElement("script");
    script.src = "https://www.youtube.com/iframe_api";
    document.head.appendChild(script);
  }, [ytReady]);

  /**
   * Create the YT player once we have all three prerequisites:
   * 1. YT API loaded (ytReady)
   * 2. User has clicked the play overlay (userClicked) — satisfies autoplay policy
   * 3. We have a video ID to load (pendingVideoId) — avoids empty player error
   *
   * The player is created with the video ID and autoplay=1 so it starts
   * immediately. Subsequent videos are loaded via player.loadVideoById().
   */
  useEffect(() => {
    if (!ytReady || !userClicked || !pendingVideoId || !playerContainerRef.current || playerRef.current) return;

    // Capture the video ID before clearing — the player constructor needs it
    // but we must not call setState synchronously inside the effect.
    const videoId = pendingVideoId;

    playerRef.current = new window.YT.Player(playerContainerRef.current, {
      width: "100%",
      height: "100%",
      videoId,
      playerVars: {
        autoplay: 1,
        controls: 0,
        modestbranding: 1,
        rel: 0,
        showinfo: 0,
      },
      events: {
        // onReady fires asynchronously once the player is fully initialized —
        // this is an external system callback, so setState here is safe.
        onReady: () => {
          setPlayerCreated(true);
        },
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
  }, [ytReady, userClicked, pendingVideoId, sessionId, viewerToken]);

  /**
   * Maps incoming SSE events to YouTube player actions:
   *  - "play"        → resume playback (or set pending video if player not yet created)
   *  - "pause"       → pause playback
   *  - "load"        → load a new video by ID and start playing
   *  - "queue:ended" → show the end-of-queue overlay
   */
  const handleSSEEvent = useCallback(
    (event: SSEEvent) => {
      switch (event.type) {
        case "load": {
          videoEndedFiredRef.current = false;
          setQueueEnded(false);
          const player = playerRef.current;
          if (player) {
            // Player already exists — load the new video directly
            player.loadVideoById(event.videoId);
            player.playVideo();
          } else {
            // Player not created yet — store the video ID so the player
            // creation effect picks it up once the user clicks the overlay
            setPendingVideoId(event.videoId);
          }
          break;
        }
        case "play": {
          const player = playerRef.current;
          if (player) {
            player.playVideo();
          }
          break;
        }
        case "pause": {
          const player = playerRef.current;
          if (player) {
            player.pauseVideo();
          }
          break;
        }
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

  // Dismisses the play overlay, satisfying the browser's autoplay gesture
  // requirement. The player will be created once we also have a video ID
  // (either already pending from an SSE event, or arriving shortly after).
  const handleStart = () => {
    setUserClicked(true);
  };

  // Show the overlay until the user has clicked AND the player has been created
  const showOverlay = !userClicked || (!playerCreated && !pendingVideoId);

  return (
    <div className="relative h-screen w-screen bg-black">
      {/* Player container — YT.Player will be injected here */}
      <div className="flex h-full w-full items-center justify-center">
        <div className="relative w-full" style={{ aspectRatio: "16/9" }}>
          <div ref={playerContainerRef} className="h-full w-full" />

          {/* Marquee overlays */}
          {config && <PlayerOverlay config={config} queue={queue} />}
        </div>
      </div>

      {/* Play button overlay — shown until user clicks to satisfy autoplay policy */}
      {showOverlay && (
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-4 bg-black/80">
          <button
            onClick={handleStart}
            className="rounded-full bg-white/10 p-8 transition-colors hover:bg-white/20"
          >
            <Play size={64} className="text-white" />
          </button>
          {userClicked && !pendingVideoId && (
            <p className="text-sm text-zinc-400">
              Waiting for admin to start playback...
            </p>
          )}
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
