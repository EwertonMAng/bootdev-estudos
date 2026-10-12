// Package auth contains the authentication helpers used across the application,
// including password hashing, JWT generation, and validation of incoming bearer tokens.
package auth

import (
	"crypto/hmac"     // Provides the HMAC API used by PBKDF2-SHA256.
	"crypto/rand"     // Generates cryptographically secure salts and refresh tokens.
	"crypto/sha256"   // Implements SHA-256, which is part of the password derivation flow.
	"crypto/subtle"   // Compares hashes using constant-time logic to reduce timing leaks.
	"encoding/base64" // Encodes and decodes the PBKDF2 components stored in the hash string.
	"encoding/hex"    // Converts random bytes to a hexadecimal refresh-token format.
	"errors"          // Provides reusable error values for malformed authorization headers.
	"fmt"             // Formats error output and hash strings.
	"net/http"        // Allows reading the Authorization header from HTTP requests.
	"strconv"         // Converts the iteration count stored in the hash into an integer.
	"strings"         // Splits the hash format and validates the bearer-token prefix.
	"time"            // Controls token expiration windows and JWT timestamps.

	"github.com/golang-jwt/jwt/v5" // JWT library used for access-token creation and validation.
	"github.com/google/uuid"       // Generates and parses UUID values to identify users.
)

const (
	passwordIterations = 600_000 // Number of PBKDF2 iterations used to derive the password hash.
	passwordSaltSize   = 16      // Random salt size in bytes, making each password hash unique.
)

// HashPassword creates a PBKDF2-SHA256 hash for the supplied password.
func HashPassword(password string) (string, error) {
	// Generate a unique random salt so the same password does not produce the same hash.
	salt := make([]byte, passwordSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Derive the password hash using PBKDF2 with the configured iteration count.
	hash := pbkdf2SHA256([]byte(password), salt, passwordIterations, sha256.Size)
	return fmt.Sprintf("$pbkdf2-sha256$%d$%s$%s", passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash)), nil
}

// CheckPasswordHash compares a plain-text password with a stored PBKDF2 hash.
func CheckPasswordHash(password, hash string) bool {
	// Split the stored hash into its components: algorithm, iterations, salt, and derived value.
	parts := strings.Split(hash, "$")
	if len(parts) != 5 || parts[1] != "pbkdf2-sha256" {
		return false
	}

	// Ensure the iteration value is valid before using it for verification.
	iterations, err := strconv.Atoi(parts[2])
	if err != nil || iterations < 1 || iterations > 10_000_000 {
		return false
	}

	// Decode the stored salt and the expected hash value from base64.
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(salt) == 0 {
		return false
	}

	expected, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(expected) == 0 {
		return false
	}

	// Recompute the hash using the password and the stored salt/iteration count.
	actual := pbkdf2SHA256([]byte(password), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// pbkdf2SHA256 implements the PBKDF2-SHA256 key derivation used to protect stored passwords.
func pbkdf2SHA256(password, salt []byte, iterations, keyLength int) []byte {
	const hashLength = sha256.Size
	blocks := (keyLength + hashLength - 1) / hashLength
	result := make([]byte, 0, blocks*hashLength)

	// PBKDF2 derives a key by iteratively mixing the password, salt, and block index.
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		value := append([]byte(nil), u...)

		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range value {
				value[j] ^= u[j]
			}
		}
		result = append(result, value...)
	}
	return result[:keyLength]
}

// MakeJWT creates a signed JWT for a user with an expiration window configured by the caller.
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(tokenSecret))
}

// ValidateJWT parses an incoming JWT and returns the authenticated user ID when valid.
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	// Type assert the parsed claims to the registered-claims structure and confirm the token is valid.
	mapClaims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}

	// The user identity is stored in the subject claim, which is a UUID string.
	userID, err := uuid.Parse(mapClaims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user_id format in token subject: %v", err)
	}

	return userID, nil
}

// GetBearerToken extracts the token from the Authorization header and validates its bearer format.
func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Authorization header not found")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return "", fmt.Errorf("Authorization header must use 'Bearer' format")
	}

	return token, nil
}

// MakeRefreshToken creates a secure random value used to renew the user's session.
func MakeRefreshToken() (string, error) {
	// Reserve 32 bytes so the token has strong entropy and can be stored safely in the database.
	tokenBytes := make([]byte, 32)

	// Fill the bytes with cryptographically secure random data.
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}

	// Convert the random bytes into a hexadecimal string for easy storage and comparison.
	tokenHex := hex.EncodeToString(tokenBytes)
	return tokenHex, nil
}

// GetAPIKey extracts an ApiKey credential from the Authorization header.
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
