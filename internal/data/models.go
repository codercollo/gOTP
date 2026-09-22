// Package data provides the persistence layer and store interfaces.
package data

import "time"

// VerifyResult represents the outcome of an atomic OTP verification attempt.
type VerifyResult int

const (
	ResultOK       VerifyResult = iota // Code mathced and was consumed
	ResultNoCode                       // No active code found for phone
	ResultExpired                      // Code existed but expired
	ResultLocked                       // Exceeded maximum failed attempts
	ResultMismatch                     // Incorrect code, attempts remaining
)

// String returns the string  representation of a VerifyResult.
func (r VerifyResult) String() string {
	switch r {
	case ResultOK:
		return "ok"
	case ResultNoCode:
		return "no_code"
	case ResultExpired:
		return "expired"
	case ResultLocked:
		return "locked"
	case ResultMismatch:
		return "mismatch"
	default:
		return "unknown"
	}
}

// OTPStore defines the interface for storing and atomically verifying OTP codes.
type OTPStore interface {
	Insert(phone, hash string, ttl time.Duration, maxAttempts int)
	VerifyAndConsume(phone, code string) VerifyResult
	GC(now time.Time) int
	Len() int
}

// Models wraps data store interfaces injected into application handlers.\
type Models struct {
	OTP OTPStore
}

// NewMNodels initializes Models with default in-memory implementations.
func NewModels() Models {
	return Models{
		OTP: NewInMemoryOTPStore(),
	}
}
