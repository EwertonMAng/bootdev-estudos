-- Cria um usuário usando os quatro parâmetros recebidos pelo código Go.
-- :one informa ao sqlc que a consulta retorna exatamente uma linha.
-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;


-- name: UpdateUser :one
UPDATE users
SET email = $2, hashed_password = $3, updated_at = $4
WHERE id = $1
RETURNING id, created_at, updated_at, email, hashed_password;

-- name: DeleteUserID :one
DELETE FROM users
WHERE id = $1
RETURNING id, created_at, updated_at, email, hashed_password;

-- Remove todos os usuários; :exec indica que nenhum resultado será retornado.
-- name: DeleteUsers :exec
DELETE FROM users;