// Package otp provides secure random code generation and constant-time hashing helpers.
package otp

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"math/big"
	"time"
)

// Configuration defaults for OTP generation and validation.
const (
	DefaultLength      = 6               // Default OTP didgit length.
	DefaultTTL         = 5 * time.Minute // Default expiration period.
	DefaultMaxAttempts = 5               // Default maximum failed validation attempts.
)

// GenerateNumeric creates a cryptographicall y secure random numeric string of length n.
func GenerateNumeric(n int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, n)
	max := big.NewInt(int64(len(digits)))

	// Pick random digit for each index.
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = digits[idx.Int64()]
	}
	return string(b), nil
}

// Hash converts an OTP code into a hex-encoded SHA-256 stign for secure storage.
func Hash(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// Match performs a constant-time comparison between code and storedHahs to prevent  timing attacks.
func Match(code, storedHash string) bool {
	return subtle.ConstantTimeCompare([]byte(Hash(code)), []byte(storedHash)) == 1
}
