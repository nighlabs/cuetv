package models

// RoomConfigResponse is the full room configuration returned by GET /config.
type RoomConfigResponse struct {
	TopMarqueeEnabled    bool   `json:"topMarqueeEnabled"`
	BottomMarqueeEnabled bool   `json:"bottomMarqueeEnabled"`
	TopMarqueeLabel      string `json:"topMarqueeLabel"`
	BottomMarqueeLabel   string `json:"bottomMarqueeLabel"`
	TopMarqueeSource     string `json:"topMarqueeSource"`
	BottomMarqueeSource  string `json:"bottomMarqueeSource"`
	CurrentIndex         int    `json:"currentIndex"`
}

// UpdateRoomConfigRequest is the payload for PATCH /config. All fields are
// pointers to support partial updates: only non-nil fields are written to the
// database, leaving omitted fields unchanged.
type UpdateRoomConfigRequest struct {
	TopMarqueeEnabled    *bool   `json:"topMarqueeEnabled"`
	BottomMarqueeEnabled *bool   `json:"bottomMarqueeEnabled"`
	TopMarqueeLabel      *string `json:"topMarqueeLabel"`
	BottomMarqueeLabel   *string `json:"bottomMarqueeLabel"`
	TopMarqueeSource     *string `json:"topMarqueeSource"`
	BottomMarqueeSource  *string `json:"bottomMarqueeSource"`
}
