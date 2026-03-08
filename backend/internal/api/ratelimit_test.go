package api

import (
	"testing"
	"time"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := newRateLimiter(1*time.Minute, 3)
	defer close(rl.stop)

	for i := 0; i < 3; i++ {
		if !rl.allow("1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := newRateLimiter(1*time.Minute, 3)
	defer close(rl.stop)

	for i := 0; i < 3; i++ {
		rl.allow("1.2.3.4")
	}

	if rl.allow("1.2.3.4") {
		t.Error("4th request should be blocked")
	}
}

func TestRateLimiter_SeparateIPs(t *testing.T) {
	rl := newRateLimiter(1*time.Minute, 2)
	defer close(rl.stop)

	rl.allow("1.1.1.1")
	rl.allow("1.1.1.1")

	if rl.allow("1.1.1.1") {
		t.Error("IP 1 should be blocked")
	}
	if !rl.allow("2.2.2.2") {
		t.Error("IP 2 should still be allowed")
	}
}
