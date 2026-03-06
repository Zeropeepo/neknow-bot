CREATE TYPE message_role AS ENUM ('user', 'assistant');

CREATE TABLE IF NOT EXISTS messages (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID         NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role       message_role NOT NULL,
    content    TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_session_id ON messages(session_id);
