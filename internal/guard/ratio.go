// Package guard provides in-memory fraud detection algorithms; velocity
// rate-limiting and prefix-level conversion tracking for SMS toll fraud mitigation.
package guard

import "sync"

// Ratio monitors send-to-verify conversion ratios per phone number prefix to detect
// and throttle SMS-pumping and Artificially Inflated Traffic attacks
type Ratio struct {
	mu        sync.Mutex
	prefixLen int
	minSends  int     // minimum send attempts
	maxRatio  float64 // maximum allowed sends-per-verification ratio threshold
	sends     map[string]int
	verifies  map[string]int
}

// NewRatio constructs and initializes a new Ratio fraud detector instance.
func NewRatio(prefixLen, minSends int, maxRatio float64) *Ratio {
	return &Ratio{
		prefixLen: prefixLen,
		minSends:  minSends,
		maxRatio:  maxRatio,
		sends:     make(map[string]int),
		verifies:  make(map[string]int),
	}
}

// prefix extracts the prefix string from phone based on configured length.
func (r *Ratio) prefix(phone string) string {
	if len(phone) <= r.prefixLen {
		return phone
	}
	return phone[:r.prefixLen]
}

// Allowed evaluates whether an outbound SMS to phone is permitted under current prefix converion metrics.
func (r *Ratio) Allowed(phone string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	p := r.prefix(phone)
	s := r.sends[p]
	if s < r.minSends {
		return true // skip enforcement until sufficient sample size is reached
	}
	verifies := r.verifies[p]
	if verifies == 0 {
		return false // block when sends exist with zero verified attempts
	}

	return float64(s)/float64(verifies) <= r.maxRatio

}

// RecordSend increments the outbound SMS counter for phone's prefix.
func (r *Ratio) RecordSend(phone string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sends[r.prefix(phone)]++
}

// RecordVerify increments the successful verification counter for phone's prefix
func (r *Ratio) RecordVerify(phone string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.verifies[r.prefix(phone)]++
}

// Stats returns the  recorded send and verify counts for phone's prefix.
func (r *Ratio) Stats(phone string) (sends, verifies int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.prefix(phone)
	return r.sends[p], r.verifies[p]
}
