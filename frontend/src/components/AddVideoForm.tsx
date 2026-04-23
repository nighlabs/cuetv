import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { addToQueue } from "@/services/api";
import { isValidYouTubeUrl } from "@/services/youtubeValidation";
import { useSessionStore } from "@/stores/sessionStore";

/**
 * Form for adding a YouTube video to the queue. URL validation is performed
 * client-side for immediate UX feedback (enable/disable submit), but the
 * backend is the authority for extracting and validating the video ID.
 */
export function AddVideoForm() {
  const [url, setUrl] = useState("");
  const [marqueeText, setMarqueeText] = useState("");
  const sessionId = useSessionStore((s) => s.sessionId);
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () => addToQueue(sessionId!, url, marqueeText),
    onSuccess: () => {
      setUrl("");
      setMarqueeText("");
      queryClient.invalidateQueries({ queryKey: ["queue", sessionId] });
    },
  });

  // Re-evaluates on every keystroke to enable/disable the submit button,
  // giving the user immediate feedback as they type or paste a URL.
  const isValid = isValidYouTubeUrl(url);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isValid) {
      mutation.mutate();
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <input
        type="text"
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        placeholder="Paste YouTube URL"
        className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-4 py-3 text-white placeholder-zinc-500 focus:border-blue-500 focus:outline-none"
      />
      <input
        type="text"
        value={marqueeText}
        onChange={(e) => setMarqueeText(e.target.value)}
        placeholder="Marquee text (optional)"
        maxLength={200}
        className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-4 py-3 text-white placeholder-zinc-500 focus:border-blue-500 focus:outline-none"
      />
      <button
        type="submit"
        disabled={!isValid || mutation.isPending}
        className="w-full rounded-md bg-blue-600 px-4 py-3 font-medium text-white hover:bg-blue-700 disabled:opacity-50"
      >
        {mutation.isPending ? "Adding..." : "Add to Queue"}
      </button>
      {mutation.isError && (
        <p className="text-sm text-red-400">Failed to add video</p>
      )}
    </form>
  );
}
