// Package data_test validates thread safety, concurrency guarantees, and lifecycle operations of the in-memory OTP store.
package data

import (
	"sync"
	"testing"
	"time"

	"github.com/codercollo/gOTP/internal/otp"
)

// TestVerifyAndConsume_ExactlyOnce verifies concurrent race safety so a code is accepted only once under high contention.
func TestVerifyAndConsume_ExactlyOnce(t *testing.T) {
	const phone = "+254700000000"
	const code = "123456"

	// Initialize store with test code
	store := NewInMemoryOTPStore()
	store.Insert(phone, otp.Hash(code), time.Minute, 1000)

	const goroutines = 200
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		okCount int
	)

	// Launch concurrent verification workers
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // Release all goroutines simultaneously for max contention
			if store.VerifyAndConsume(phone, code) == ResultOK {
				mu.Lock()
				okCount++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	// Assert code was consumed exactly once
	if okCount != 1 {
		t.Fatalf("code accepted %d times, want exactly 1", okCount)
	}

	// Verify record deletion
	if store.Len() != 0 {
		t.Errorf("store not empty after consume: %d records", store.Len())
	}
}

// TestVerifyAndConsume_Outcomes validates all verification state branches.
func TestVerifyAndConsume_Outcomes(t *testing.T) {
	const phone = "+254711111111"

	// Test lookup without active code
	t.Run("no code", func(t *testing.T) {
		s := NewInMemoryOTPStore()
		if got := s.VerifyAndConsume(phone, "000000"); got != ResultNoCode {
			t.Errorf("got %v, want no_code", got)
		}
	})

	// Test handling of expired code
	t.Run("expired", func(t *testing.T) {
		s := NewInMemoryOTPStore()
		s.Insert(phone, otp.Hash("123456"), -1*time.Second, 5)
		if got := s.VerifyAndConsume(phone, "123456"); got != ResultExpired {
			t.Errorf("got %v, want expired", got)
		}
	})

	// Test max attempts tracking and lockout behavior
	t.Run("mismatch then lock", func(t *testing.T) {
		s := NewInMemoryOTPStore()
		s.Insert(phone, otp.Hash("123456"), time.Minute, 3)
		if got := s.VerifyAndConsume(phone, "999999"); got != ResultMismatch {
			t.Errorf("attempt 1: got %v, want mismatch", got)
		}
		if got := s.VerifyAndConsume(phone, "999999"); got != ResultMismatch {
			t.Errorf("attempt 2: got %v, want mismatch", got)
		}
		if got := s.VerifyAndConsume(phone, "999999"); got != ResultLocked {
			t.Errorf("attempt 3: got %v, want locked", got)
		}
	})

	// Test successful consumption prevents reuse
	t.Run("success consumes", func(t *testing.T) {
		s := NewInMemoryOTPStore()
		s.Insert(phone, otp.Hash("123456"), time.Minute, 5)
		if got := s.VerifyAndConsume(phone, "123456"); got != ResultOK {
			t.Errorf("got %v, want ok", got)
		}
		if got := s.VerifyAndConsume(phone, "123456"); got != ResultNoCode {
			t.Errorf("second verify: got %v, want no_code", got)
		}
	})
}

// TestGC validates garbage collection of expired store records.
func TestGC(t *testing.T) {
	s := NewInMemoryOTPStore()
	s.Insert("+254700000001", otp.Hash("111111"), -1*time.Second, 5) // Expired
	s.Insert("+254700000002", otp.Hash("222222"), time.Minute, 5)    // Active

	// Assert purge removes only expired entries
	if purged := s.GC(time.Now()); purged != 1 {
		t.Errorf("purged %d, want 1", purged)
	}
	if s.Len() != 1 {
		t.Errorf("len %d, want 1", s.Len())
	}
}
