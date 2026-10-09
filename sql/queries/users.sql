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


-- Remove todos os usuários; :exec indica que nenhum resultado será retornado.
-- name: DeleteUsers :exec
DELETE FROM users;