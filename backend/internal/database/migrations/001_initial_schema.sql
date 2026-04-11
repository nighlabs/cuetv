CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    friend_join_key TEXT NOT NULL UNIQUE,
    viewer_token TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS room_config (
    session_id TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
    top_marquee_enabled INTEGER NOT NULL DEFAULT 0,
    bottom_marquee_enabled INTEGER NOT NULL DEFAULT 0,
    top_marquee_label TEXT NOT NULL DEFAULT '',
    bottom_marquee_label TEXT NOT NULL DEFAULT '',
    top_marquee_source TEXT NOT NULL DEFAULT 'current',
    bottom_marquee_source TEXT NOT NULL DEFAULT 'current',
    current_index INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS queue_items (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    youtube_video_id TEXT NOT NULL,
    youtube_url TEXT NOT NULL,
    marquee_text TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL,
    added_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_queue_items_session_position ON queue_items(session_id, position);
CREATE INDEX IF NOT EXISTS idx_sessions_friend_join_key ON sessions(friend_join_key);
