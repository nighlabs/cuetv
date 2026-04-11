/**
 * Matches four common YouTube URL formats and captures the 11-character
 * video ID in group 1:
 * - youtube.com/watch?v=XXXXXXXXXXX
 * - youtube.com/embed/XXXXXXXXXXX
 * - youtube.com/shorts/XXXXXXXXXXX
 * - youtu.be/XXXXXXXXXXX
 */
const YOUTUBE_REGEX =
  /^(?:https?:\/\/)?(?:www\.|m\.)?(?:youtube\.com\/(?:watch\?v=|embed\/|shorts\/)|youtu\.be\/)([a-zA-Z0-9_-]{11})(?:[?&].*)?$/;

/**
 * Extracts the 11-character YouTube video ID from a URL string.
 * Returns `null` for invalid or unrecognized URLs. This is client-side
 * UX validation only; the backend is the authority for URL validation.
 */
export function extractVideoId(url: string): string | null {
  const match = url.trim().match(YOUTUBE_REGEX);
  return match ? match[1] : null;
}

/** Convenience wrapper — returns `true` if the URL contains a valid YouTube video ID. */
export function isValidYouTubeUrl(url: string): boolean {
  return extractVideoId(url) !== null;
}

/**
 * Returns the public YouTube thumbnail URL for a given video ID.
 * Uses the `default.jpg` size (120x90px).
 */
export function getThumbnailUrl(videoId: string): string {
  return `https://img.youtube.com/vi/${videoId}/default.jpg`;
}
