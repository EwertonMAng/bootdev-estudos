package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	auth "github.com/EwertonMAng/http_clients/internal/auth"
	"github.com/EwertonMAng/http_clients/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// apiConfig reúne recursos compartilhados pelos handlers, como métricas e acesso ao banco.
type apiConfig struct {
	// fileserverHits conta acessos aos arquivos estáticos; atomic.Int32 protege atualizações concorrentes.
	fileserverHits atomic.Int32
	DB             *database.Queries // Consultas SQL tipadas geradas pelo sqlc.
	platform       string            // Ambiente da aplicação, usado para proteger operações de desenvolvimento.
	tokenSecret    string            // Chave secreta usada para assinar tokens JWT.
}

// User define os campos de usuário que a API envia nas respostas JSON.
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

// middlewareMetricsInc envolve um handler para contar cada requisição que passa por ele.
// O contador é atômico, então pode ser atualizado com segurança por várias requisições simultâneas.
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// handlerMetrics mostra uma página HTML simples com a quantidade de visitas ao servidor de arquivos.
func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	htmlTemplate := `
  <br/>    Welcome, Chirpy Admin<br/>    Chirpy has been visited %d times!<br/>  <br/>`
	body := fmt.Sprintf(htmlTemplate, cfg.fileserverHits.Load())
	w.Write([]byte(body))
}

// handlerReset zera as métricas e apaga os usuários, mas permite isso somente no ambiente de desenvolvimento.
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	// Impede uma operação destrutiva em ambientes diferentes de "dev".
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Reset is only allowed in dev environment."))
		return
	}

	// Reinicia o contador antes de solicitar ao banco que remova os usuários.
	cfg.fileserverHits.Store(0)

	// O contexto da requisição permite cancelar a operação se a conexão do cliente for encerrada.
	err := cfg.DB.DeleteUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete users")
		return
	}

	// Um status 200 sem corpo indica que a operação terminou com sucesso.
	w.WriteHeader(http.StatusOK)
}

// readinessHandler é um endpoint de verificação: responde OK quando o servidor está atendendo requisições.
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// respondWithJSON serializa payload como JSON e envia o código HTTP informado.
func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Os cabeçalhos precisam ser definidos antes de escrever o status e o corpo da resposta.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

// respondWithError padroniza respostas de erro usando o formato JSON {"error": "..."}.
func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

// cleanChirp substitui palavras proibidas por asteriscos sem alterar a capitalização das outras palavras.
func cleanChirp(text string) string {
	badWords := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}

	// Divide o texto em palavras; strings.Split mantém os espaços como separadores entre os itens.
	words := strings.Split(text, " ")

	for i, word := range words {
		// A busca no mapa fica indiferente a maiúsculas e minúsculas.
		loweredWord := strings.ToLower(word)

		// A posição original é preservada para remontar a frase na mesma ordem.
		if badWords[loweredWord] {
			words[i] = "****"
		}
	}

	// Reúne as palavras em uma string, separando-as novamente por espaços.
	return strings.Join(words, " ")
}

// handlerUsersCreate cria um usuário a partir de um e-mail enviado no corpo JSON da requisição.
func (cfg *apiConfig) handlerUsersCreate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	// Primeiro converte o JSON recebido para uma struct tipada.
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash password")
		return
	}

	// Usa o mesmo instante UTC para criação e atualização, e gera um identificador único para o usuário.
	now := time.Now().UTC()
	user, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		ID:             uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create user")
		return
	}

	// Converte o modelo gerado pelo SQLC para User, que define os campos expostos na resposta JSON.
	respondWithJSON(w, http.StatusCreated, User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	})
}

func (cfg *apiConfig) handlerCrudChirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	type Chirp struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	// 1. Validar e extrair o token Bearer do cabeçalho ANTES de tudo
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// 2. Validar o JWT usando o segredo armazenado na configuração
	userID, err := auth.ValidateJWT(bearerToken, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// 3. Decodificar o corpo da requisição
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleaned := cleanChirp(params.Body)

	now := time.Now().UTC()
	chirp, err := cfg.DB.CreateChirps(r.Context(), database.CreateChirpsParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Body:      cleaned,
		UserID:    userID, // Utiliza o ID extraído de dentro do token JWT validado
	})

	if err != nil {
		log.Printf("Erro ao criar chirp no banco: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't create a chirp")
		return
	}

	respondWithJSON(w, http.StatusCreated, Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	type Chirp struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	// Chama a query que retorna todos os chirps ordenados por data (ascendente)
	dbChirps, err := cfg.DB.GetChirpsAsc(r.Context())
	if err != nil {
		log.Printf("Erro ao buscar chirps no banco: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't get chirps")
		return
	}

	// Mapeia os dados do banco para a struct pública de resposta
	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		})
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerGetChirpID(w http.ResponseWriter, r *http.Request) {
	type Chirp struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	// 1. Pega o ID diretamente pelo PathValue definido na rota
	idString := r.PathValue("id")

	// 2. Converte para UUID
	parsedID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	// 3. Busca no banco
	dbChirp, err := cfg.DB.GetChirpByID(r.Context(), parsedID)
	if err != nil {
		// Se o erro for exatamente porque não encontrou o registro no banco:
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}

		// Para qualquer outro erro de banco, retorna 500
		log.Printf("Erro ao buscar chirp no banco: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't get chirp")
		return
	}

	// 4. Retorna o chirp encontrado com sucesso (200 OK)
	respondWithJSON(w, http.StatusOK, Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	})
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		ExpiresIn int64  `json:"expires_in_seconds"` // Duração em segundos
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	user, err := cfg.DB.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			// Correção aqui:
			respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}
		log.Printf("Erro ao buscar usuário no banco: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't get user")
		return
	}

	if !auth.CheckPasswordHash(params.Password, user.HashedPassword) {
		// Correção aqui:
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	expirationTime := time.Hour
	if params.ExpiresIn > 0 {
		duration := time.Duration(params.ExpiresIn) * time.Second
		if duration < expirationTime {
			expirationTime = duration
		}
	}

	accessToken, err := auth.MakeJWT(user.ID, cfg.tokenSecret, expirationTime)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create access token")
		return
	}

	type response struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		Token     string    `json:"token"`
	}

	respondWithJSON(w, http.StatusOK, response{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     accessToken,
	})
}

// handlerGetChirps busca todos os chirps no banco e retorna os dados em JSON.

// handlerGetChirpID busca um chirp pelo ID fornecido na URL e retorna os dados em JSON.

// main prepara as dependências da aplicação e inicia o servidor HTTP.
func main() {
	// Carrega configurações locais do arquivo .env para o ambiente do processo.
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erro ao carregar o arquivo .env")
	}

	// Lê as configurações usadas para conectar ao banco e identificar o ambiente atual.
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	tokenSecret := os.Getenv("TOKEN_SECRET")

	// Abre a conexão com PostgreSQL. O driver é registrado pelo import com "_" acima.
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Não foi possível conectar ao banco:", err)
	}
	// Garante que a conexão será fechada quando main terminar.
	defer db.Close()

	// Cria a camada de consultas tipadas gerada pelo sqlc, usada pelos handlers.
	dbQueries := database.New(db)

	// Compartilha configuração e acesso ao banco entre os handlers que precisam deles.
	apiCfg := &apiConfig{
		DB:          dbQueries,
		platform:    platform,
		tokenSecret: tokenSecret,
	}

	// O mux associa cada método e caminho HTTP ao handler responsável pela rota.
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", readinessHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCrudChirps)
	mux.HandleFunc("POST /api/users", apiCfg.handlerUsersCreate)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{id}", apiCfg.handlerGetChirpID)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	// Serve os arquivos estáticos na rota /app/ e remove esse prefixo antes de procurar o arquivo.
	// O middleware incrementa a métrica apenas nas requisições que passam por esse servidor de arquivos.
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))

	// Configura o servidor para receber requisições na porta 8080 usando as rotas do mux.
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Servidor rodando na porta :8080...")
	// ListenAndServe bloqueia enquanto o servidor estiver ativo; um erro encerra a aplicação.
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
