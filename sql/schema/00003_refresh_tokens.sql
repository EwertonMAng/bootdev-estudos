-- Create the refresh tokens table that supports session renewal without user re-login.
-- +goose Up
CREATE TABLE refresh_tokens (
    token VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

-- Add an index to speed up lookups by user when validating refresh tokens.
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Undo the migration by removing the refresh_tokens table and its index.
-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
