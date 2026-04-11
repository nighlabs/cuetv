-- name: CreateRoomConfig :exec
INSERT INTO room_config (session_id, top_marquee_enabled, bottom_marquee_enabled, top_marquee_label, bottom_marquee_label, top_marquee_source, bottom_marquee_source, current_index)
VALUES (?, 0, 0, '', '', 'current', 'current', 0);

-- name: GetRoomConfig :one
SELECT session_id, top_marquee_enabled, bottom_marquee_enabled, top_marquee_label, bottom_marquee_label, top_marquee_source, bottom_marquee_source, current_index
FROM room_config
WHERE session_id = ?;

-- name: UpdateRoomConfig :exec
UPDATE room_config
SET top_marquee_enabled = ?,
    bottom_marquee_enabled = ?,
    top_marquee_label = ?,
    bottom_marquee_label = ?,
    top_marquee_source = ?,
    bottom_marquee_source = ?
WHERE session_id = ?;

-- name: UpdateCurrentIndex :exec
UPDATE room_config SET current_index = ? WHERE session_id = ?;

-- name: GetCurrentIndex :one
SELECT current_index FROM room_config WHERE session_id = ?;
