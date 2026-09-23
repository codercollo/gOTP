// Package guard_test provides unit tests for velocity rate limiting, ratio-based fraud protection,
// and composite guard logic.
package guard

import (
	"sync"
	"testing"
	"time"
)

// TestVelocity_ExactBurstUnderConcurrency verifies that concurrent requests accurately exhaust
// capacity without state corruption or race conditions.
func TestVelocity_ExactBurstUnderConcurrency(t *testing.T) {
	const capacity = 5
	v := NewVelocity(capacity, 0) // disable token replenishment
	now := time.Now()

	const goroutines = 200
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
		start   = make(chan struct{})
	)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if v.Allow("+254700000000", now) {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	if allowed != capacity {
		t.Fatalf("allowed %d, want exactly %d", allowed, capacity)
	}
}

// TestVelocity_Refill verifies token bucket replenishment over elapsed time intervals.
func TestVelocity_Refill(t *testing.T) {
	v := NewVelocity(1, 1) // 1 initial token, 1 token/sec refill
	t0 := time.Now()

	if !v.Allow("k", t0) {
		t.Fatal("first event should be allowed")
	}
	if v.Allow("k", t0) {
		t.Fatal("second immediate event should be blocked")
	}
	if !v.Allow("k", t0.Add(time.Second)) {
		t.Fatal("event after 1s refill should be allowed")
	}
}

// TestRatio_ThrottlesPumping verifies that high send-to-verify ratios trigger throttling on suspicious prefixes.
func TestRatio_ThrottlesPumping(t *testing.T) {
	r := NewRatio(6, 10, 5.0) // activate after 10 sends, threshold ratio of 5.0
	const phone = "+254700123456"

	// Simulate 20 sends with zero verifications to exceed the threshold
	for i := 0; i < 20; i++ {
		r.RecordSend(phone)
	}
	if r.Allowed(phone) {
		t.Fatal("prefix flooded with sends and no verifies should be blocked")
	}

	// Record sufficient verifications to lower ratio back within permitted limits
	for i := 0; i < 10; i++ {
		r.RecordVerify(phone)
	}
	if !r.Allowed(phone) {
		t.Fatal("healthy send/verify ratio should be allowed")
	}
}

// TestRatio_AllowsBeforeMinSends verifies that throttling rules remain dormant until reaching minSends.
func TestRatio_AllowsBeforeMinSends(t *testing.T) {
	r := NewRatio(6, 10, 5.0)
	r.RecordSend("+254700123456")
	if !r.Allowed("+254700123456") {
		t.Fatal("should allow before minSends reached")
	}
}

// TestGuard_ExactCountsUnderBurst verifies concurrent execution handling through the unified Guard interface.
func TestGuard_ExactCountsUnderBurst(t *testing.T) {
	g := New(Config{
		VelocityCapacity: 10,
		VelocityRefill:   0,
		RatioPrefixLen:   6,
		RatioMinSends:    1_000_000, // set high threshold to isolate velocity check
		RatioMax:         5,
	})
	now := time.Now()

	const goroutines = 300
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
		start   = make(chan struct{})
	)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if g.CheckSend("+254700000000", "", now).Allowed {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	if allowed != 10 {
		t.Fatalf("allowed %d, want exactly 10 (velocity capacity)", allowed)
	}
}
