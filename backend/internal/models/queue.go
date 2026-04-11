package models

// AddQueueItemRequest is the payload for adding a video to the session queue.
// URL must be a valid YouTube URL; the video ID is extracted server-side.
// MarqueeText is the optional display text shown in the viewer marquee.
type AddQueueItemRequest struct {
	URL         string `json:"url"`
	MarqueeText string `json:"marqueeText"`
}

// ReorderQueueRequest is the payload for reordering the session queue.
// Order must be the complete ordered list of all queue item IDs; items not
// included will be removed and the positions of all items are reassigned to
// match the new ordering.
type ReorderQueueRequest struct {
	Order []string `json:"order"`
}

// QueueItemResponse represents a single video in the session queue as returned
// by the API.
type QueueItemResponse struct {
	ID             string `json:"id"`
	YouTubeVideoID string `json:"youtubeVideoId"`
	YouTubeURL     string `json:"youtubeUrl"`
	MarqueeText    string `json:"marqueeText"`
	Position       int    `json:"position"`
	AddedAt        string `json:"addedAt"`
}
