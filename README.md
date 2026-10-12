# http_clients

[![Go version](https://img.shields.io/badge/Go-1.27.1-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-not%20specified-lightgrey?style=flat)](#licen%C3%A7a-e-contribui%C3%A7%C3%A3o)
[![Build](https://img.shields.io/badge/build-not%20configured-lightgrey?style=flat)](#testes-e-qualidade)
[![Go Report Card](https://goreportcard.com/badge/github.com/EwertonMAng/http_clients)](https://goreportcard.com/report/github.com/EwertonMAng/http_clients)

API HTTP em Go para cadastro e autenticação de usuários, publicação de chirps e renovação de sessões usando PostgreSQL.

## Visão geral

O `http_clients` é um serviço web que expõe uma API JSON para gerenciamento de usuários e chirps. O servidor usa o roteador padrão `net/http`, persiste dados em PostgreSQL e oferece autenticação com senhas derivadas por PBKDF2-SHA256, access tokens JWT e refresh tokens armazenados no banco.

O fluxo principal é:

1. O servidor carrega configurações de um arquivo `.env` na raiz do repositório.
2. Abre uma conexão PostgreSQL e prepara as consultas tipadas geradas pelo sqlc.
3. Registra as rotas HTTP e inicia o servidor na porta `8080`.
4. Os handlers validam e decodificam as requisições, executam operações no banco e respondem em JSON.
5. O middleware da rota `/app/` contabiliza os acessos aos arquivos estáticos.

### Estrutura do repositório

```text
.
├── assets/                 # Recursos estáticos do projeto
├── internal/
│   ├── auth/               # Hash de senha, JWT, refresh tokens e validação de credenciais
│   └── database/           # Camada de acesso ao PostgreSQL gerada pelo sqlc
├── sql/
│   ├── queries/             # Consultas-fonte usadas pelo sqlc
│   └── schema/              # Migrações de banco de dados compatíveis com Goose
├── go.mod                  # Caminho do módulo e dependências Go
├── go.sum                  # Checksums das dependências
├── main.go                 # Configuração do servidor e handlers HTTP
├── sqlc.yaml               # Configuração de geração do sqlc
└── index.html              # Página servida como arquivo estático
```

O código executável está no pacote raiz (`main.go`); este repositório não possui diretórios `cmd/`, `pkg/`, `api/` ou `configs/`. O conteúdo de `internal/` é privado ao módulo segundo as regras de importação de pacotes Go. Os arquivos `internal/database/*.sql.go` são gerados: altere `sql/queries/` ou `sql/schema/` e regenere-os com sqlc, em vez de editar os arquivos gerados manualmente.

## Funcionalidades

- Cadastro de usuários e atualização do próprio e-mail e senha.
- Hash de senhas com PBKDF2-SHA256 e salt aleatório.
- Login com access token JWT de uma hora e refresh token válido por 60 dias.
- Renovação de access token e revogação de refresh token.
- Criação, listagem, consulta e exclusão de chirps.
- Filtro de chirps por autor (`author_id`) e ordenação ascendente ou descendente (`sort`).
- Limpeza das palavras `kerfuffle`, `sharbert` e `fornax` no conteúdo dos chirps.
- Webhook autenticado por API key para habilitar Chirpy Red para um usuário.
- Endpoint de prontidão, página de métricas administrativas e reset restrito ao ambiente `dev`.
- Servimento dos arquivos locais sob `/app/`.

### Endpoints HTTP

Todas as rotas da API estão registradas em `main.go`. Endpoints que recebem objetos JSON esperam `Content-Type: application/json`. Erros de API normalmente usam o formato `{"error":"..."}`.

| Método | Caminho | Autenticação | Descrição |
| --- | --- | --- | --- |
| `GET` | `/api/healthz` | Não | Verificação de prontidão; responde `200 OK`. |
| `POST` | `/api/users` | Não | Cria usuário. Recebe `email` e `password`; responde `201 Created`. |
| `PUT` | `/api/users` | Bearer access token | Atualiza e-mail e senha do usuário autenticado. |
| `POST` | `/api/login` | Não | Autentica usuário e retorna access token e refresh token. |
| `POST` | `/api/refresh` | Bearer refresh token | Emite um novo access token. |
| `POST` | `/api/revoke` | Bearer refresh token | Revoga o refresh token; responde `204 No Content`. |
| `POST` | `/api/chirps` | Bearer access token | Cria chirp, com limite de 140 bytes. |
| `GET` | `/api/chirps` | Não | Lista chirps; aceita `author_id` e `sort=asc\|desc`. |
| `GET` | `/api/chirps/{id}` | Não | Consulta um chirp pelo UUID. |
| `DELETE` | `/api/chirps/{id}` | Bearer access token | Exclui chirp somente se pertencer ao usuário autenticado. |
| `POST` | `/api/polka/webhooks` | `ApiKey` | Recebe eventos de upgrade de assinatura. |
| `GET` | `/admin/metrics` | Não | Exibe em HTML a contagem de acessos aos arquivos em `/app/`. |
| `POST` | `/admin/reset` | Não; somente `PLATFORM=dev` | Zera métricas e remove usuários. |
| `GET` | `/app/` | Não | Serve arquivos estáticos do diretório do projeto. |

## Stack tecnológica

| Componente | Tecnologia | Uso |
| --- | --- | --- |
| Runtime | Go `1.27.1` (declarado em `go.mod`) | Linguagem e runtime da aplicação. |
| HTTP | Biblioteca padrão `net/http` | Servidor, roteamento e handlers; sem framework externo. |
| Persistência | `database/sql` | API padrão para acesso ao banco relacional. |
| Banco de dados | PostgreSQL | Armazenamento de usuários, chirps e refresh tokens. |
| Driver PostgreSQL | `github.com/lib/pq` | Driver registrado para `database/sql`. |
| Geração de queries | sqlc | Gera consultas Go tipadas a partir de `sql/queries/` e `sql/schema/`. |
| Autenticação | `github.com/golang-jwt/jwt/v5` | Assinatura e validação de access tokens JWT. |
| UUID | `github.com/google/uuid` | Identificadores de usuários e chirps. |
| Configuração local | `github.com/joho/godotenv` | Carrega variáveis do arquivo `.env`. |

## Pré-requisitos

- Go `1.27.1` ou compatível com a versão declarada no módulo.
- Uma instância PostgreSQL acessível pela aplicação.
- Um banco criado e as migrações de `sql/schema/` aplicadas.
- `sqlc` somente se for necessário regenerar a camada de consultas.
- Goose CLI ou outro processo de migração compatível somente para aplicar as migrações. O repositório contém diretivas Goose, mas não inclui o executável de migração.

## Instalação e execução

Clone o repositório e baixe as dependências:

```bash
git clone https://github.com/EwertonMAng/http_clients.git
cd http_clients
go mod download
```

`go mod tidy` pode ser usado para reconciliar o manifesto e os checksums durante o desenvolvimento:

```bash
go mod tidy
```

Crie o banco PostgreSQL e aplique, em ordem numérica, as migrações de `sql/schema/`. Por exemplo, se Goose já estiver instalado e configurado para o seu banco:

```bash
goose -dir sql/schema postgres 'postgres://USER:PASSWORD@HOST:5432/DATABASE?sslmode=disable' up
```

Crie um arquivo `.env` na raiz. O carregamento desse arquivo é obrigatório no início do servidor; a aplicação encerra se não conseguir carregá-lo.

```dotenv
DB_URL=postgres://USER:PASSWORD@HOST:5432/DATABASE?sslmode=disable
PLATFORM=dev
TOKEN_SECRET=replace-with-a-long-random-secret
POLKA_KEY=replace-with-your-webhook-api-key
```

`DB_URL` deve conter uma URL de conexão válida para seu PostgreSQL. `TOKEN_SECRET` é usado para assinar access tokens e `POLKA_KEY` autentica o webhook. Não use os valores ilustrativos em produção nem versione o arquivo `.env`. Defina `PLATFORM` como `dev` apenas em ambientes de desenvolvimento: nesse modo, `/admin/reset` pode remover os usuários.

Inicie o servidor a partir da raiz:

```bash
go run .
```

O servidor escuta em `:8080`. Para compilar um binário otimizado para distribuição:

```bash
mkdir -p bin
go build -ldflags="-s -w" -o bin/http_clients .
```

Não há `Makefile`, `docker-compose.yml`, `Dockerfile` nem workflow de CI neste repositório no momento; portanto, não existem alvos `make` ou comandos Docker definidos pelo projeto.

## Testes e qualidade

Execute a suíte com cobertura e detector de condições de corrida:

```bash
go test -v -race -cover ./...
```

O repositório contém testes unitários para o pacote de autenticação. Também é possível executar a suíte simples com:

```bash
go test ./...
```

Não há benchmarks definidos no momento. O comando abaixo pode ser usado quando forem adicionados testes `Benchmark...`:

```bash
go test -bench=. ./...
```

`golangci-lint` não está configurado como dependência nem como CI do projeto. Caso esteja instalado no ambiente, pode ser executado manualmente:

```bash
golangci-lint run
```

Formate mudanças em Go antes de submetê-las:

```bash
gofmt -w main.go internal/auth/*.go internal/database/*.go
```

## Exemplo de uso

Os exemplos abaixo assumem que o servidor está em execução em `http://localhost:8080`.

### Verificar prontidão

```bash
curl -i http://localhost:8080/api/healthz
```

Resposta esperada:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8

OK
```

### Criar usuário e autenticar

```bash
curl -i http://localhost:8080/api/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct-horse-battery-staple"}'
```

Resposta `201 Created` (UUID e timestamps são gerados pelo servidor):

```json
{
  "id": "UUID-GERADO",
  "created_at": "TIMESTAMP-UTC",
  "updated_at": "TIMESTAMP-UTC",
  "email": "ada@example.com",
  "is_chirpy_red": false
}
```

Faça login para obter os tokens. A resposta também contém o ID, timestamps, e-mail e estado de Chirpy Red do usuário:

```bash
curl -i http://localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct-horse-battery-staple"}'
```

O corpo da resposta `200 OK` inclui `token` e `refresh_token`. Copie os valores recebidos para os exemplos protegidos abaixo:

```bash
ACCESS_TOKEN='cole-aqui-o-token'
REFRESH_TOKEN='cole-aqui-o-refresh-token'
```

### Criar e listar chirps

```bash
curl -i http://localhost:8080/api/chirps \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"body":"Hello from Chirpy!"}'
```

A criação responde `201 Created` com `id`, `created_at`, `updated_at`, `body` e `user_id`. Palavras bloqueadas são substituídas por `****`.

```bash
curl -i 'http://localhost:8080/api/chirps?sort=desc'
```

Para filtrar por autor, use um UUID válido em `author_id`:

```bash
curl -i 'http://localhost:8080/api/chirps?author_id=UUID-DO-USUARIO&sort=asc'
```

Consulte ou exclua um chirp substituindo o marcador pelo ID retornado na criação:

```bash
curl -i http://localhost:8080/api/chirps/UUID-DO-CHIRP
curl -i -X DELETE http://localhost:8080/api/chirps/UUID-DO-CHIRP \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

A exclusão retorna `204 No Content` quando o chirp pertence ao usuário autenticado; retorna `403 Forbidden` se pertencer a outro usuário.

### Atualizar perfil, renovar ou revogar sessão

```bash
curl -i -X PUT http://localhost:8080/api/users \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada.new@example.com","password":"another-secure-password"}'
```

```bash
curl -i http://localhost:8080/api/refresh \
  -H "Authorization: Bearer $REFRESH_TOKEN"
```

A renovação responde `200 OK` com um novo campo `token`. Para invalidar o refresh token:

```bash
curl -i -X POST http://localhost:8080/api/revoke \
  -H "Authorization: Bearer $REFRESH_TOKEN"
```

### Webhook de upgrade e métricas administrativas

O webhook espera o esquema abaixo; substitua a chave pela variável `POLKA_KEY` configurada no servidor:

```bash
curl -i -X POST http://localhost:8080/api/polka/webhooks \
  -H 'Authorization: ApiKey replace-with-your-webhook-api-key' \
  -H 'Content-Type: application/json' \
  -d '{"event":"user.upgraded","data":{"user_id":"UUID-DO-USUARIO"}}'
```

Eventos diferentes de `user.upgraded` são aceitos sem alteração e respondem `204 No Content`. Consulte as métricas de acesso a arquivos com:

```bash
curl -i http://localhost:8080/admin/metrics
```

O endpoint de reset remove usuários e dados relacionados por cascata, além de zerar o contador de arquivos; ele só funciona com `PLATFORM=dev`:

```bash
curl -i -X POST http://localhost:8080/admin/reset
```

## Docker

O repositório não contém artefatos Docker ou configuração Compose. Assim, não há comandos oficiais para construção de imagem ou inicialização de containers definidos pelo projeto. A aplicação pode ser executada diretamente com `go run .` após configurar PostgreSQL e `.env`.

## Licença e contribuição

Não foi encontrado arquivo de licença no repositório. Até que uma licença seja adicionada e publicada, os termos de uso e redistribuição não estão especificados neste projeto.

Contribuições devem manter as convenções existentes e limitar mudanças ao escopo proposto. Antes de abrir um pull request:

1. Formate o código Go com `gofmt`.
2. Execute `go test -v -race -cover ./...`.
3. Para alterações no banco, atualize as migrações ou consultas-fonte em `sql/` e regenere o código sqlc, em vez de editar manualmente os arquivos gerados.
4. Descreva as alterações e inclua exemplos ou testes relevantes.
