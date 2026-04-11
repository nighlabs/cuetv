-- name: AddQueueItem :exec
INSERT INTO queue_items (id, session_id, youtube_video_id, youtube_url, marquee_text, position, added_at)
VALUES (?, ?, ?, ?, ?, ?, datetime('now'));

-- name: GetQueueItems :many
SELECT id, session_id, youtube_video_id, youtube_url, marquee_text, position, added_at
FROM queue_items
WHERE session_id = ?
ORDER BY position ASC;

-- name: GetQueueItem :one
SELECT id, session_id, youtube_video_id, youtube_url, marquee_text, position, added_at
FROM queue_items
WHERE id = ? AND session_id = ?;

-- name: GetQueueItemAtPosition :one
SELECT id, session_id, youtube_video_id, youtube_url, marquee_text, position, added_at
FROM queue_items
WHERE session_id = ? AND position = ?;

-- name: GetQueueItemCount :one
SELECT COUNT(*) FROM queue_items WHERE session_id = ?;

-- name: UpdateQueueItemPosition :exec
UPDATE queue_items SET position = ? WHERE id = ? AND session_id = ?;

-- name: UpdateQueueItemMarqueeText :exec
UPDATE queue_items SET marquee_text = ? WHERE id = ?;

-- name: DeleteQueueItem :exec
DELETE FROM queue_items WHERE id = ? AND session_id = ?;
