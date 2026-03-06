CREATE TABLE IF NOT EXISTS chat_sessions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id     UUID        NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_sessions_bot_id  ON chat_sessions(bot_id);
CREATE INDEX idx_chat_sessions_user_id ON chat_sessions(user_id);
