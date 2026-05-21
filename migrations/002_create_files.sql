CREATE TABLE IF NOT EXISTS files (
    id         BIGSERIAL PRIMARY KEY,
    filename   TEXT NOT NULL,
    path       TEXT NOT NULL,
    size       BIGINT NOT NULL DEFAULT 0,
    mime_type  TEXT NOT NULL DEFAULT 'application/octet-stream',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
