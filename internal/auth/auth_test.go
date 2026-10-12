// Package auth contains the authentication tests for JWT creation and validation.
package auth

import (
	"testing" // Provides the Go test framework used to validate the auth helpers.
	"time"    // Supplies durations for testing expiration logic.

	"github.com/google/uuid" // Creates unique user IDs during the test scenarios.
)

// TestJWT validates successful token creation, successful validation, and rejection of invalid tokens.
func TestJWT(t *testing.T) {
	// Create a unique user ID for the test scenario.
	userID := uuid.New()
	tokenSecret := "secret-key" // Use a simple secret for the test environment.
	expiresIn := time.Hour      // Tokens are valid for one hour in this test.

	// Create a token that should be valid and accepted by the validator.
	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("failed to make jwt: %v", err)
	}

	// Validate the newly created token and ensure it belongs to the expected user.
	parsedID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Fatalf("failed to validate jwt: %v", err)
	}

	if parsedID != userID {
		t.Errorf("expected user id %v, got %v", userID, parsedID)
	}

	// An invalid secret should not validate a legitimate token.
	_, err = ValidateJWT(tokenString, "wrong-secret")
	if err == nil {
		t.Error("expected error when validating with wrong secret, got nil")
	}

	// A token created with a negative expiration should be rejected as expired.
	expiredToken, err := MakeJWT(userID, tokenSecret, -time.Hour)
	if err != nil {
		t.Fatalf("failed to make expired jwt: %v", err)
	}

	_, err = ValidateJWT(expiredToken, tokenSecret)
	if err == nil {
		t.Error("expected error when validating expired token, got nil")
	}
}
