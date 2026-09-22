// Package data provides in-memory OTP storage and atomic verification operations.
package data

import (
	"sync"
	"time"

	"github.com/codercollo/gOTP/internal/otp"
)

// otpRecord represents an active OTP entry stored in memory.
type otpRecord struct {
	hash        string
	expiresAt   time.Time
	attempts    int
	maxAttempts int
}

// InMemoryOTPStore is a thread-safe map-backed implementation of OTPStore
type InMemoryOTPStore struct {
	mu sync.Mutex // Guards map access
	m  map[string]*otpRecord
}

// NewInMemoryOTPStore contructs a ready-to-use InMemoryOTPStore.
func NewInMemoryOTPStore() *InMemoryOTPStore {
	return &InMemoryOTPStore{m: make(map[string]*otpRecord)}
}

// Insert create or replaces an OTP record for a phone number.
func (s *InMemoryOTPStore) Insert(phone, hash string, ttl time.Duration, maxAttempts int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.m[phone] = &otpRecord{
		hash:        hash,
		expiresAt:   time.Now().Add(ttl),
		attempts:    0,
		maxAttempts: maxAttempts,
	}

}

// VerifyAndConsume atomically validates an OTP code and deletes it on success or lockout.
func (s *InMemoryOTPStore) VerifyAndConsume(phone, code string) VerifyResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if record exists
	rec, ok := s.m[phone]
	if !ok {
		return ResultNoCode
	}

	// Purge and return if code expired
	if time.Now().After(rec.expiresAt) {
		delete(s.m, phone)
		return ResultExpired
	}

	// Purge and return if locked out
	if rec.attempts >= rec.maxAttempts {
		delete(s.m, phone)
		return ResultLocked
	}

	// Consume and return success on match
	if otp.Match(code, rec.hash) {
		delete(s.m, phone)
		return ResultOK
	}

	// Increment failed attempt count
	rec.attempts++
	if rec.attempts >= rec.maxAttempts {
		delete(s.m, phone)
		return ResultLocked
	}

	return ResultMismatch

}

// GC purges all expired OTP entries relative to now.
func (s *InMemoryOTPStore) GC(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	purged := 0
	for phone, rec := range s.m {
		if now.After(rec.expiresAt) {
			delete(s.m, phone)
			purged++
		}
	}
	return purged
}

// Len returns the count of active entries in the store.
func (s *InMemoryOTPStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.m)
}
