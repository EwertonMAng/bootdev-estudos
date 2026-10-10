-- Migração de subida: cria a tabela que armazena os usuários da aplicação.
-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email TEXT NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL,
    is_chirpy_red BOOLEAN NOT NULL DEFAULT FALSE
);

-- Migração de reversão: permite desfazer a criação da tabela.
-- +goose Down
DROP TABLE users;