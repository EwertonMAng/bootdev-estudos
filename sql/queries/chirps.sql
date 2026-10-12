-- Create a new chirp and return the saved row to the API layer.
-- :one tells sqlc that the query returns exactly one row.
-- name: CreateChirps :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- Delete a single chirp by its UUID.
-- name: DeleteChirpID :exec
DELETE FROM chirps
WHERE id = $1;

-- Fetch all chirps ordered from oldest to newest.
-- name: GetChirpsAsc :many
SELECT * FROM chirps
ORDER BY created_at ASC;

-- Retrieve one chirp by its primary key for detail pages and ownership validation.
-- name: GetChirpByID :one
SELECT * FROM chirps
WHERE id = $1;

-- Retrieve chirps authored by a specific user, ordered chronologically.
-- name: GetChirpsByAuthor :many
SELECT * FROM chirps
WHERE user_id = $1
ORDER BY created_at ASC;
