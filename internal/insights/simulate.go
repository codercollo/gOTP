package insights

import (
	"context"
	"strings"
)

// MockChecker simulates SIM-swap status for deterministic testing and demos.
type MockChecker struct {
	swapped map[string]bool
}

// NewMock creates a MockChecker initialized with numbers to treat as swapped.
func NewMock(numbers []string) *MockChecker {
	m := &MockChecker{
		swapped: make(map[string]bool),
	}
	for _, n := range numbers {
		n = strings.TrimSpace(n)
		if n != "" {
			m.swapped[n] = true
		}

	}
	return m
}

// RecentSwap checks if the phone number is marked as swapped.
func (m *MockChecker) RecentSwap(ctx context.Context, phone string) (bool, error) {
	return m.swapped[phone], nil
}
