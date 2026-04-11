import { useMutation } from "@tanstack/react-query";
import { sendPlaybackCommand } from "@/services/api";
import { useSessionStore } from "@/stores/sessionStore";
import { Play, Pause, SkipBack, SkipForward } from "lucide-react";

/**
 * Playback control bar that sends play/pause/next/prev commands to the
 * backend. All buttons use a 44px minimum touch target to meet mobile
 * usability guidelines for the phone-based admin interface.
 */
export function PlaybackControls() {
  const sessionId = useSessionStore((s) => s.sessionId);

  const mutation = useMutation({
    mutationFn: (command: "play" | "pause" | "next" | "prev") =>
      sendPlaybackCommand(sessionId!, command),
  });

  const buttonClass =
    "min-h-[44px] min-w-[44px] rounded-md bg-zinc-800 p-3 text-white hover:bg-zinc-700 disabled:opacity-50 border border-zinc-700";

  return (
    <div className="flex items-center justify-center gap-3">
      <button
        onClick={() => mutation.mutate("prev")}
        disabled={mutation.isPending}
        className={buttonClass}
      >
        <SkipBack size={24} />
      </button>
      <button
        onClick={() => mutation.mutate("play")}
        disabled={mutation.isPending}
        className={buttonClass}
      >
        <Play size={24} />
      </button>
      <button
        onClick={() => mutation.mutate("pause")}
        disabled={mutation.isPending}
        className={buttonClass}
      >
        <Pause size={24} />
      </button>
      <button
        onClick={() => mutation.mutate("next")}
        disabled={mutation.isPending}
        className={buttonClass}
      >
        <SkipForward size={24} />
      </button>
    </div>
  );
}
