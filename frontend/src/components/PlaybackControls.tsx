import { useMutation } from "@tanstack/react-query";
import { sendPlaybackCommand } from "@/services/api";
import { useSessionStore } from "@/stores/sessionStore";
import { useSocketStore } from "@/stores/socketStore";
import { Play, Pause, SkipBack, SkipForward } from "lucide-react";

/**
 * Playback control bar that sends play/pause/next/prev commands to the
 * backend. All buttons use a 44px minimum touch target to meet mobile
 * usability guidelines. The active state (playing/paused) is tracked
 * locally and highlighted on the corresponding button.
 */
export function PlaybackControls() {
  const sessionId = useSessionStore((s) => s.sessionId);
  const { playbackState, setPlaybackState } = useSocketStore();

  const mutation = useMutation({
    mutationFn: (command: "play" | "pause" | "next" | "prev") =>
      sendPlaybackCommand(sessionId!, command),
    onSuccess: (_data, command) => {
      switch (command) {
        case "play":
        case "next":
        case "prev":
          setPlaybackState("playing");
          break;
        case "pause":
          setPlaybackState("paused");
          break;
      }
    },
  });

  const baseClass =
    "min-h-[44px] min-w-[44px] rounded-md p-3 disabled:opacity-50 border transition-colors";

  const buttonClass = (active: boolean) =>
    active
      ? `${baseClass} bg-blue-600 text-white border-blue-500 hover:bg-blue-700`
      : `${baseClass} bg-zinc-800 text-white border-zinc-700 hover:bg-zinc-700`;

  return (
    <div className="flex items-center justify-center gap-3">
      <button
        onClick={() => mutation.mutate("prev")}
        disabled={mutation.isPending}
        className={buttonClass(false)}
      >
        <SkipBack size={24} />
      </button>
      <button
        onClick={() => mutation.mutate("play")}
        disabled={mutation.isPending}
        className={buttonClass(playbackState === "playing")}
      >
        <Play size={24} />
      </button>
      <button
        onClick={() => mutation.mutate("pause")}
        disabled={mutation.isPending}
        className={buttonClass(playbackState === "paused")}
      >
        <Pause size={24} />
      </button>
      <button
        onClick={() => mutation.mutate("next")}
        disabled={mutation.isPending}
        className={buttonClass(false)}
      >
        <SkipForward size={24} />
      </button>
    </div>
  );
}
