// package auth centraliza as operações relacionadas à autenticação, geração de hash e JWT.
package auth

import (
	"crypto/hmac"     // Fornece a API HMAC exigida para a derivação PBKDF2-SHA256.
	"crypto/rand"     // Gera salt e tokens aleatórios seguros.
	"crypto/sha256"   // Implementa SHA-256, usado na derivação de senha e no JWT.
	"crypto/subtle"   // Compara hashes em tempo constante para evitar vazamento de informação.
	"encoding/base64" // Codifica e decodifica os componentes do hash em base64.
	"fmt"             // Formata mensagens de erro e serialização de saída.
	"strconv"         // Converte strings numéricas em inteiros para os parâmetros do hash.
	"strings"         // Auxilia ao parse do formato de hash e do cabeçalho Authorization.
	"time"            // Gerencia expiração e emissão de JWTs.

	"encoding/hex" // Converte bytes em representação hexadecimal para refresh tokens.
	"errors"       // Fornece tipos de erro para validação de cabeçalhos e tokens.
	"net/http"     // Permite ler o cabeçalho Authorization das requisições HTTP.

	"github.com/golang-jwt/jwt/v5" // Biblioteca JWT para geração e validação de tokens.
	"github.com/google/uuid"       // Gera e interpreta UUIDs associados aos usuários.
)

const (
	passwordIterations = 600_000 // Número de iterações usado na derivação PBKDF2 para senha.
	passwordSaltSize   = 16      // Tamanho do salt em bytes para tornar o hash único por usuário.
)

// HashPassword gera um hash PBKDF2-SHA256 de uma senha.
func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltSize)     // Cria um salt aleatório para aumentar a segurança do hash.
	if _, err := rand.Read(salt); err != nil { // Verifica falha na geração do salt.
		return "", err // Retorna erro para o chamador.
	}
	hash := pbkdf2SHA256([]byte(password), salt, passwordIterations, sha256.Size) // Deriva o hash usando PBKDF2.
	return fmt.Sprintf("$pbkdf2-sha256$%d$%s$%s", passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash)), nil // Retorna o formato do hash com iterações, salt e valor derivado.
}

// CheckPasswordHash compara uma senha em texto com o hash armazenado.
func CheckPasswordHash(password, hash string) bool {
	parts := strings.Split(hash, "$")                   // Separa o hash em partes conforme o formato $pbkdf2-sha256$iterations$salt$hash.
	if len(parts) != 5 || parts[1] != "pbkdf2-sha256" { // Verifica se a estrutura do hash é válida.
		return false // Se o formato for inválido, rejeita a senha.
	}
	iterations, err := strconv.Atoi(parts[2])                    // Converte o número de iterações armazenado no hash.
	if err != nil || iterations < 1 || iterations > 10_000_000 { // Garante que o valor de iterações tenha um intervalo razoável.
		return false // Rejeita dados que não fazem sentido ou são potencialmente maliciosos.
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3]) // Decodifica o salt salvo na string do hash.
	if err != nil || len(salt) == 0 {                         // Verifica se o salt existe e é válido.
		return false // Se o salt falhar, a senha não pode ser validada.
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[4]) // Decodifica o hash esperado armazenado.
	if err != nil || len(expected) == 0 {                         // Verifica se o valor esperado foi decodificado corretamente.
		return false // Rejeita hash inválido.
	}
	actual := pbkdf2SHA256([]byte(password), salt, iterations, len(expected)) // Recalcula o hash da senha informada usando salt e iterações do hash armazenado.
	return subtle.ConstantTimeCompare(actual, expected) == 1                  // Compara em tempo constante para evitar timing attacks.
}

// pbkdf2SHA256 implementa a derivação PBKDF2-SHA256 usada no armazenamento seguro de senhas.
func pbkdf2SHA256(password, salt []byte, iterations, keyLength int) []byte {
	const hashLength = sha256.Size                      // Tamanho do digest SHA-256 em bytes.
	blocks := (keyLength + hashLength - 1) / hashLength // Calcula quantos blocos são necessários para produzir a chave final.
	result := make([]byte, 0, blocks*hashLength)        // Prealoca espaço para receber os bytes do hash final.
	for block := 1; block <= blocks; block++ {          // Processa cada bloco do derivado da chave.
		mac := hmac.New(sha256.New, password)                                                         // Cria um HMAC-SHA256 usando a senha como chave.
		_, _ = mac.Write(salt)                                                                        // Acrescenta o salt ao HMAC.
		_, _ = mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)}) // Acrescenta o índice do bloco para a derivação PBKDF2.
		u := mac.Sum(nil)                                                                             // Gera o valor inicial U1.
		value := append([]byte(nil), u...)                                                            // Copia U1 para o valor acumulado.
		for i := 1; i < iterations; i++ {                                                             // Reaproveita a lógica de PBKDF2 para cada iteração.
			mac = hmac.New(sha256.New, password) // Reinicia HMAC para calcular a próxima iteração.
			_, _ = mac.Write(u)                  // Usa o valor anterior como entrada do próximo cálculo.
			u = mac.Sum(nil)                     // Gera o novo bloco U_i.
			for j := range value {               // XOR em cada byte do valor acumulado com U_i.
				value[j] ^= u[j] // Combina os resultados de todas as iterações.
			}
		}
		result = append(result, value...) // Acumula os bytes gerados pelo bloco atual.
	}
	return result[:keyLength] // Retorna apenas o número de bytes exigidos pela chave final.
}

// MakeJWT gera um JWT assinado para um usuário específico com tempo de expiração configurado.
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{ // Cria as reivindicações padrão do JWT.
		Issuer:    "chirpy-access",                                     // Nome do emissor do token.
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),                // Tempo em que o token foi emitido.
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()), // Tempo de expiração do token.
		Subject:   userID.String(),                                     // Armazena o identificador do usuário no subject.
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // Cria o token usando algoritmo HMAC-SHA256.
	return token.SignedString([]byte(tokenSecret))             // Assina o JWT com a chave secreta fornecida.
}

// ValidateJWT valida um JWT recebido e retorna o UUID do usuário caso ele seja legítimo.
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{} // Estrutura que receberá as claims do token.

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) { // Faz parse e valida assinatura do JWT.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { // Garante que o algoritmo utilizado seja HMAC.
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"]) // Retorna erro se houver algoritmo inesperado.
		}
		return []byte(tokenSecret), nil // Usa a chave secreta configurada para validar a assinatura.
	})

	if err != nil { // Verifica falha no parse do token ou na assinatura.
		return uuid.Nil, err // Retorna UUID nulo e o erro original.
	}

	// Alternativa mais segura verificando o tipo diretamente do token.Claims.
	mapClaims, ok := token.Claims.(*jwt.RegisteredClaims) // Faz type assertion para claims registradas do JWT.
	if !ok || !token.Valid {                              // Confirma o tipo e validade do token.
		return uuid.Nil, fmt.Errorf("invalid token") // Responde com erro quando o token é inválido.
	}

	userID, err := uuid.Parse(mapClaims.Subject) // Converte o subject em UUID para obter o usuário.
	if err != nil {                              // Verifica se o subject não contém um UUID válido.
		return uuid.Nil, fmt.Errorf("invalid user_id format in token subject: %v", err) // Retorna erro do formato do subject.
	}

	return userID, nil // Retorna o ID do usuário autenticado.
}

// GetBearerToken extrai e valida o valor do cabeçalho Authorization no formato Bearer <token>.
func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization") // Lê o cabeçalho Authorization da requisição.
	if authHeader == "" {                      // Verifica ausência do cabeçalho.
		return "", fmt.Errorf("Authorization header not found") // Retorna erro explícito.
	}

	token := strings.TrimPrefix(authHeader, "Bearer ") // Remove o prefixo Bearer para obter somente o token.
	if token == authHeader {                           // Se o prefixo não estiver presente, o token não é válido.
		return "", fmt.Errorf("Authorization header must use ******") // Retorna erro de formato.
	}

	return token, nil // Retorna o token puro para uso posterior.
}

// MakeRefreshToken cria um identificador aleatório e seguro para renovar a sessão do usuário.
func MakeRefreshToken() (string, error) {
	// Cria um slice de 32 bytes para armazenar um token aleatório de 256 bits.
	tokenBytes := make([]byte, 32)

	// Preenche o slice com dados aleatórios seguros criptograficamente.
	_, err := rand.Read(tokenBytes)
	if err != nil { // Verifica falha ao gerar bytes aleatórios.
		return "", err // Retorna erro ao chamador.
	}

	// Converte os bytes para uma string hexadecimal.
	tokenHex := hex.EncodeToString(tokenBytes)

	return tokenHex, nil // Retorna um refresh token em formato hexadecimal.
}

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no authorization header included")
	}

	splitAuth := strings.Split(authHeader, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "ApiKey" {
		return "", errors.New("malformed authorization header")
	}

	return splitAuth[1], nil
}
