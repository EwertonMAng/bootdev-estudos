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

-- Remove todos os usuários; :exec indica que nenhum resultado será retornado.
-- name: DeleteChirps :exec
DELETE FROM chirps;

-- name: GetChirpsAsc :many
SELECT * FROM chirps
ORDER BY created_at ASC;

-- name: GetChirpByID :one 
SELECT * FROM chirps
WHERE id = $1; 