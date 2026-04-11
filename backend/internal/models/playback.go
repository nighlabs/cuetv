package models

// PlaybackCommandRequest is the payload for issuing a playback command.
// Valid Command values are "play", "pause", "next", and "prev".
type PlaybackCommandRequest struct {
	Command string `json:"command"`
}
