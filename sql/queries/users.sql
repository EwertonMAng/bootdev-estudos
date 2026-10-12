-- Create a new user row and return the created record.
-- :one tells sqlc that the query returns exactly one row.
-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password, is_chirpy_red)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- Lookup a user by email address so the application can validate credentials.
-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- Update the user's profile details, including the hashed password.
-- name: UpdateUser :one
UPDATE users
SET email = $2, hashed_password = $3, updated_at = $4
WHERE id = $1
RETURNING id, created_at, updated_at, email, hashed_password, is_chirpy_red;

-- Delete a single user by unique identifier.
-- name: DeleteUserID :one
DELETE FROM users
WHERE id = $1
RETURNING id, created_at, updated_at, email, hashed_password, is_chirpy_red;

-- Remove all users; this is used by the development reset endpoint.
-- :exec indicates the query does not return rows.
-- name: DeleteUsers :exec
DELETE FROM users;

-- Fetch one user by primary key for authorization and profile reads.
-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- Promote a user to the Chirpy Red subscription tier.
-- name: UpdateUserToChirpyRed :one
UPDATE users
SET is_chirpy_red = TRUE, updated_at = $2
WHERE id = $1
RETURNING id, created_at, updated_at, email, hashed_password, is_chirpy_red;
