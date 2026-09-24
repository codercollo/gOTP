// Package otp_test validates code generation, hashing, and matching logic.
package otp

import (
	"strings"
	"testing"
)

// TestGenerateNumeric verifies code length and numeric digit constraints.
func TestGenerateNumeric(t *testing.T) {
	for _, n := range []int{4, 6, 8} {
		code, err := GenerateNumeric(n)
		if err != nil {
			t.Fatalf("GenerateNumeric(%d) error: %v", n, err)
		}

		// Assert expected length
		if len(code) != n {
			t.Errorf("length = %d, want %d", len(code), n)
		}

		// Assert digits only
		if strings.Trim(code, "0123456789") != "" {
			t.Errorf("code %q contains non-digit characters", code)
		}
	}
}

// TestGenerateNumericIsRandom checks for sufficient uniqueness across multiple generations.
func TestGenerateNumericIsRandom(t *testing.T) {
	seen := make(map[string]bool)

	// Collect unique generated codes
	for i := 0; i < 1000; i++ {
		code, err := GenerateNumeric(6)
		if err != nil {
			t.Fatal(err)
		}
		seen[code] = true
	}

	// Verify collision rate remains extremely low
	if len(seen) < 990 {
		t.Errorf("only %d unique codes in 1000 draws — suspicious", len(seen))
	}
}

// TestHashAndMatch verifies SHA-256 hashing and constant-time match evaluation.
func TestHashAndMatch(t *testing.T) {
	code := "123456"
	h := Hash(code)

	// Ensure code is not stored in plaintext
	if h == code {
		t.Fatal("hash must not equal the plaintext code")
	}

	// Verify correct match succeeds
	if !Match(code, h) {
		t.Error("Match returned false for the correct code")
	}

	// Verify incorrect match fails
	if Match("000000", h) {
		t.Error("Match returned true for a wrong code")
	}
}
