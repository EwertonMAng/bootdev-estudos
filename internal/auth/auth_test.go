package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "secret-key"
	expiresIn := time.Hour

	// Testa a criação do token
	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("failed to make jwt: %v", err)
	}

	// Testa a validação bem-sucedida do token
	parsedID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Fatalf("failed to validate jwt: %v", err)
	}

	if parsedID != userID {
		t.Errorf("expected user id %v, got %v", userID, parsedID)
	}

	// Testa rejeição com secret incorreto
	_, err = ValidateJWT(tokenString, "wrong-secret")
	if err == nil {
		t.Error("expected error when validating with wrong secret, got nil")
	}

	// Testa rejeição com token expirado
	expiredToken, err := MakeJWT(userID, tokenSecret, -time.Hour)
	if err != nil {
		t.Fatalf("failed to make expired jwt: %v", err)
	}

	_, err = ValidateJWT(expiredToken, tokenSecret)
	if err == nil {
		t.Error("expected error when validating expired token, got nil")
	}
}
