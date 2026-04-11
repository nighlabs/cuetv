-- name: CreateSession :exec
INSERT INTO sessions (id, friend_join_key, viewer_token, created_at)
VALUES (?, ?, ?, datetime('now'));

-- name: GetSession :one
SELECT id, friend_join_key, viewer_token, created_at
FROM sessions
WHERE id = ?;

-- name: GetSessionByFriendKey :one
SELECT id, friend_join_key, viewer_token, created_at
FROM sessions
WHERE LOWER(friend_join_key) = LOWER(?);

-- name: GetSessionByViewerToken :one
SELECT id, friend_join_key, viewer_token, created_at
FROM sessions
WHERE viewer_token = ?;

-- name: FriendKeyExists :one
SELECT COUNT(*) > 0 AS key_exists
FROM sessions
WHERE LOWER(friend_join_key) = LOWER(?);
