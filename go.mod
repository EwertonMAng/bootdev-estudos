// Identificador do módulo: também é o caminho usado por outros pacotes para importar este projeto.
module github.com/EwertonMAng/http_clients

// Versão mínima da linguagem Go exigida pelo projeto.
go 1.27.1

// Bibliotecas usadas pelo servidor: UUIDs, leitura de .env e driver PostgreSQL.
require (
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.12.3
)

require github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
