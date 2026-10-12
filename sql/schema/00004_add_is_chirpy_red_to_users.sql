-- Migration to add the premium Chirpy Red flag to existing users.
-- +goose Up
ALTER TABLE users
ADD COLUMN IF NOT EXISTS is_chirpy_red BOOLEAN NOT NULL DEFAULT FALSE;

-- Reverse the migration by removing the premium flag if needed.
-- +goose Down
ALTER TABLE users
DROP COLUMN IF EXISTS is_chirpy_red;
