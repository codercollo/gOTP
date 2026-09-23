// Package guard provides composite fraud preventions rules combining velocity
// and conversion-ratio evaluation for SMS send requests.
package guard

import "time"

// Decision represents the result of a fraud inspection check.
type Decision struct {
	Allowed bool
	Reason  string
}

// Config specifies initialization parameters for velocity and ratio check engines.
type Config struct {
	VelocityCapacity float64 // Max token capacity for velocity limiters
	VelocityRefill   float64 // Token refill rate per second
	RatioPrefixLen   int     // Phone refill length for conversion tracking
	RatioMinSends    int     // Minimum sends required to trigger conversion
	RatioMax         float64 // max allowed send-to-verify ratio before block

	// Fan-out detection settings. A floor <= 0 deisables the check.
	FanOutWindow time.Duration
	FanOutFloor  int
	FanOutFactor float64
	FanOutAlpha  float64
}

// Guard evaluates outbound request rules using velocity, conversion-ratio, fan-out checks engines.
type Guard struct {
	vel    *Velocity
	ratio  *Ratio
	fanout *FanOut
}

// New constructs a Guard instance with velocity and ratio tracking initializes from cfg.
func New(cfg Config) *Guard {
	return &Guard{
		vel:    NewVelocity(cfg.VelocityCapacity, cfg.VelocityRefill),
		ratio:  NewRatio(cfg.RatioPrefixLen, cfg.RatioMinSends, cfg.RatioMax),
		fanout: NewFanOut(cfg.FanOutWindow, cfg.FanOutFloor, cfg.FanOutFactor, cfg.FanOutAlpha),
	}
}

// CheckSend evaluates rate limits and fraud ratios before recording a send event.
func (g *Guard) CheckSend(phone, ip string, now time.Time) Decision {
	// Limit send velocity for the destination phone.
	if !g.vel.Allow(phone, now) {
		return Decision{
			Allowed: false,
			Reason:  "velocity_phone",
		}
	}

	// Limit send velocity for the requesting source IP.
	if ip != "" && !g.vel.Allow(ip, now) {
		return Decision{
			Allowed: false,
			Reason:  "velocity_ip",
		}
	}

	// Block prefixes with an excessive send-to-verify ratio.
	if !g.ratio.Allowed(phone) {
		return Decision{
			Allowed: false,
			Reason:  "verify_ratio",
		}
	}

	// Detect one source spraying many distinct destination numbers
	if ip != "" {
		if _, blocked := g.fanout.Observe(ip, phone, now); blocked {
			return Decision{
				Allowed: false,
				Reason:  "fanout_spray",
			}
		}
	}
	g.ratio.RecordSend(phone)
	return Decision{
		Allowed: true,
	}
}

// RecordVerify registers a successful verification event for phone's prefix.
func (g *Guard) RecordVerify(phone string) {
	g.ratio.RecordVerify(phone)
}

// Stats returns the active send and verify counts for phone's prefix.
func (g *Guard) Stats(phone string) (sends, verifies int) {
	return g.ratio.Stats(phone)
}
