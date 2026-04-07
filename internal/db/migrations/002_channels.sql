-- +goose Up
CREATE TABLE IF NOT EXISTS channels (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    config TEXT NOT NULL,
    welcome_message TEXT NOT NULL DEFAULT '',
    connect_url TEXT,
    enabled INTEGER NOT NULL DEFAULT 1,
    state TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS channel_contacts (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    external_user_id TEXT NOT NULL,
    external_chat_id TEXT NOT NULL,
    username TEXT,
    display_name TEXT,
    connection_code TEXT,
    code_expires_at DATETIME,
    connected_at DATETIME,
    last_message_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(channel_id, external_user_id)
);

CREATE INDEX IF NOT EXISTS idx_channels_enabled ON channels(enabled);
CREATE INDEX IF NOT EXISTS idx_channel_contacts_channel ON channel_contacts(channel_id);
CREATE INDEX IF NOT EXISTS idx_channel_contacts_code ON channel_contacts(connection_code);

-- +goose Down
DROP INDEX IF EXISTS idx_channel_contacts_code;
DROP INDEX IF EXISTS idx_channel_contacts_channel;
DROP INDEX IF EXISTS idx_channels_enabled;
DROP TABLE IF EXISTS channel_contacts;
DROP TABLE IF EXISTS channels;
