// Package guard provides lightweight abuse and traffic-detection guards.
package guard

import (
	"sync"
	"time"
)

// FanOut detects sources contacting many distinct destination numbers in a
// short window, catching distributed spray patterns that per-key limits miss.
type FanOut struct {
	mu      sync.Mutex
	window  time.Duration
	floor   int
	factor  float64
	alpha   float64
	sources map[string]*fanSource
}

// fanSource tracks distinct destinations and the souce's historical baseline.
type fanSource struct {
	seen        map[string]bool
	ewma        float64
	windowStart time.Time
}

// NewFanOut creates a fan-out detector with the given widow and thresholds.
//
// A floor  <= 0 disables the detector.
func NewFanOut(window time.Duration, floor int, factor, alpha float64) *FanOut {
	return &FanOut{
		window:  window,
		floor:   floor,
		factor:  factor,
		alpha:   alpha,
		sources: make(map[string]*fanSource),
	}
}

// Enable reports whether the detector is active.
func (f *FanOut) Enabled() bool {
	return f != nil && f.floor > 0
}

// Observe records a destination for source and reports the current distinct
// count and whether the source should be blocked.
func (f *FanOut) Observe(source, number string, now time.Time) (distinct int, blocked bool) {
	// Ignore inactive detectors and empty sources.
	if !f.Enabled() || source == "" {
		return 0, false
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Create state for a new source.
	s, ok := f.sources[source]
	if !ok {
		s = &fanSource{
			seen:        make(map[string]bool),
			windowStart: now,
		}
		f.sources[source] = s
	}

	// Roll the window and update the historical baseline.
	if now.Sub(s.windowStart) >= f.window {
		s.ewma = f.alpha*float64(len(s.seen)) + (1-f.alpha)*s.ewma
		s.seen = make(map[string]bool)
		s.windowStart = now
	}

	// Record the destination and calculate the distinct count.
	s.seen[number] = true
	distinct = len(s.seen)

	// Use the larger of the fixed floor and adaptive threshold.
	threshold := float64(f.floor)
	if adaptive := s.ewma * f.factor; adaptive > threshold {
		threshold = adaptive
	}

	blocked = float64(distinct) > threshold
	return distinct, blocked
}
