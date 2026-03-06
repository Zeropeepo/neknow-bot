CREATE TYPE file_status AS ENUM ('pending', 'processing', 'ready', 'failed');

CREATE TABLE IF NOT EXISTS files (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id      UUID        NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    filename    TEXT        NOT NULL,
    minio_path  TEXT        NOT NULL,
    file_size   BIGINT      NOT NULL DEFAULT 0,
    mime_type   TEXT        NOT NULL DEFAULT 'text/plain',
    status      file_status NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_files_bot_id ON files(bot_id);
CREATE INDEX idx_files_status ON files(status);
