// Package insights defines interfaces and mock implementations for SIM-swap verification
// before dispatching OTP codes.
package insights

import "context"

// Checker evalutes whether a phone number experienced a recent SIM-swap event.
type Checker interface {
	RecentSwap(ctx context.Context, phone string) (bool, error)
}

// NoopChecker provides a default implementation that always reports no sim swap activity.
type NoopChecker struct{}

// RecentSwap always returns false to bypass checks without error.
func (NoopChecker) RecentSwap(ctx context.Context, phone string) (bool, error) {
	return false, nil
}
