package guard

import (
	"strconv"
	"testing"
	"time"
)

// TestFanOut_BlocksSprayFromOneSource verifies that a source targeting many
// distinct numbers within one window is blocked.
func TestFanOut_BlocksSprayFromOneSource(t *testing.T) {
	f := NewFanOut(time.Minute, 5, 3.0, 0.3)
	now := time.Now()
	const ip = "41.90.1.1"

	blocked := 0
	for i := 0; i < 30; i++ {
		if _, b := f.Observe(ip, "+2547000000"+strconv.Itoa(1000+i), now); b {
			blocked++
		}
	}

	if blocked == 0 {
		t.Fatal("spraying 30 distinct numbers from one source should be blocked")
	}
}

// TestFanOut_AllowsRepeatToSameNumber verifies that repeated requests to one
// destination do not increase the distinct count or trigger the detector.
func TestFanOut_AllowsRepeatToSameNumber(t *testing.T) {
	f := NewFanOut(time.Minute, 5, 3.0, 0.3)
	now := time.Now()
	const ip = "41.90.1.2"

	for i := 0; i < 50; i++ {
		if _, b := f.Observe(ip, "+254700000001", now); b {
			t.Fatal("repeated sends to one number must not trip the fan-out detector")
		}
	}
}

// TestFanOut_WindowExpiry verifies that destinations from an expired window
// no longer contribute to the current distinct count.
func TestFanOut_WindowExpiry(t *testing.T) {
	f := NewFanOut(time.Minute, 5, 100.0, 0.3)
	base := time.Now()
	const ip = "41.90.1.3"

	for i := 0; i < 4; i++ {
		f.Observe(ip, "+25470000000"+strconv.Itoa(i), base)
	}

	later := base.Add(2 * time.Minute)
	blocked := false
	for i := 4; i < 8; i++ {
		if _, b := f.Observe(ip, "+25470000000"+strconv.Itoa(i), later); b {
			blocked = true
		}
	}

	if blocked {
		t.Fatal("distinct numbers outside the window must not count toward the fan-out")
	}
}

// TestFanOut_DisabledWhenFloorZero verifies that a zero floor disables the
// detector and causes all observations to be allowed.
func TestFanOut_DisabledWhenFloorZero(t *testing.T) {
	f := NewFanOut(time.Minute, 0, 3.0, 0.3)

	if f.Enabled() {
		t.Fatal("floor 0 should disable the detector")
	}
	if _, b := f.Observe("ip", "num", time.Now()); b {
		t.Fatal("disabled detector must always allow")
	}
}
