-- Create the users table that stores authentication data for the application.
-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email TEXT NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL,
    is_chirpy_red BOOLEAN NOT NULL DEFAULT FALSE
);

-- Reverse the migration by removing the users table if the schema needs to be rolled back.
-- +goose Down
DROP TABLE users;
