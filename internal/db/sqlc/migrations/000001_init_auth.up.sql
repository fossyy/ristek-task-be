BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
                       id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
                       email         TEXT        NOT NULL UNIQUE,
                       password_hash TEXT        NOT NULL,
                       created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
                       updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
                                id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
                                user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                token      TEXT        NOT NULL UNIQUE,
                                expires_at TIMESTAMPTZ NOT NULL,
                                created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

COMMIT;