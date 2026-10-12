-- Create the chirps table that stores the application posts written by users.
-- +goose Up
CREATE TABLE chirps (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    body TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- Roll back the migration by deleting the chirps table.
-- +goose Down
DROP TABLE chirps;
