-- Migração de subida: cria a tabela que armazena os usuários da aplicação.
-- +goose Up
CREATE TABLE chirps (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    body TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- Migração de reversão: permite desfazer a criação da tabela.
-- +goose Down
DROP TABLE chirps;