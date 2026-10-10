// package auth concentra os testes de autenticação e geração de JWT do projeto.
package auth

import (
	"testing" // Permite escrever e executar testes Go para as funções de autenticação.
	"time" // Fornece duração para testar expiração de token.

	"github.com/google/uuid" // Gera UUIDs para simular usuários durante os testes.
)

// TestJWT valida a criação, a validação e a rejeição de tokens JWT.
func TestJWT(t *testing.T) {
	userID := uuid.New() // Cria um identificador único para o usuário de teste.
	tokenSecret := "secret-key" // Usa uma chave simples para os cenários de teste.
	expiresIn := time.Hour // Define duração de 1 hora para o access token.

	// Testa a criação do token.
	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil { // Verifica se a criação do JWT falhou.
		t.Fatalf("failed to make jwt: %v", err) // Para o teste com mensagem de erro clara.
	}

	// Testa a validação bem-sucedida do token.
	parsedID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil { // Verifica falha ao validar o token novo.
		t.Fatalf("failed to validate jwt: %v", err) // Interrompe o teste em caso de erro.
	}

	if parsedID != userID { // Confirma que o identificador do usuário retornado corresponde ao esperado.
		t.Errorf("expected user id %v, got %v", userID, parsedID) // Reporta divergência de usuário.
	}

	// Testa rejeição com secret incorreto.
	_, err = ValidateJWT(tokenString, "wrong-secret")
	if err == nil { // Verifica se a validação com chave errada foi rejeitada.
		t.Error("expected error when validating with wrong secret, got nil") // Reporta falha de segurança.
	}

	// Testa rejeição com token expirado.
	expiredToken, err := MakeJWT(userID, tokenSecret, -time.Hour) // Gera um token já expirado.
	if err != nil { // Verifica falha ao criar token expirado.
		t.Fatalf("failed to make expired jwt: %v", err) // Interrompe o teste em caso de erro.
	}

	_, err = ValidateJWT(expiredToken, tokenSecret) // Tenta validar o token expirado.
	if err == nil { // Confirma que o token expirado foi rejeitado.
		t.Error("expected error when validating expired token, got nil") // Reporta falha de expiração.
	}
}
