package services

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// videoIDRegex validates that a YouTube video ID is exactly 11 characters long
// and contains only alphanumeric characters, hyphens, and underscores. YouTube
// video IDs are always 11 characters — this constraint rejects truncated or
// malformed IDs early.
var videoIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

// ExtractYouTubeVideoID parses a YouTube URL and extracts the 11-character
// video ID. It supports four URL formats:
//   - Standard watch URLs: https://www.youtube.com/watch?v=VIDEO_ID
//   - Short URLs: https://youtu.be/VIDEO_ID
//   - Embed URLs: https://www.youtube.com/embed/VIDEO_ID
//   - Shorts URLs: https://www.youtube.com/shorts/VIDEO_ID
//
// Bare video IDs without a URL are intentionally rejected — all input must be
// a full URL so the server validates the source domain rather than trusting
// arbitrary client-supplied IDs.
func ExtractYouTubeVideoID(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme == "" {
		parsed, err = url.Parse("https://" + rawURL)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
	}

	host := strings.ToLower(parsed.Hostname())

	var videoID string

	switch {
	case host == "youtu.be":
		// https://youtu.be/VIDEO_ID
		videoID = strings.TrimPrefix(parsed.Path, "/")

	case host == "www.youtube.com" || host == "youtube.com" || host == "m.youtube.com":
		path := parsed.Path

		switch {
		case strings.HasPrefix(path, "/watch"):
			// https://www.youtube.com/watch?v=VIDEO_ID
			videoID = parsed.Query().Get("v")

		case strings.HasPrefix(path, "/embed/"):
			// https://www.youtube.com/embed/VIDEO_ID
			videoID = strings.TrimPrefix(path, "/embed/")

		case strings.HasPrefix(path, "/shorts/"):
			// https://www.youtube.com/shorts/VIDEO_ID
			videoID = strings.TrimPrefix(path, "/shorts/")

		default:
			return "", fmt.Errorf("unsupported YouTube URL format")
		}

	default:
		return "", fmt.Errorf("not a YouTube URL")
	}

	// Strip any trailing path or query from videoID
	if idx := strings.IndexAny(videoID, "/?&"); idx != -1 {
		videoID = videoID[:idx]
	}

	if !videoIDRegex.MatchString(videoID) {
		return "", fmt.Errorf("invalid YouTube video ID: %q", videoID)
	}

	return videoID, nil
}
