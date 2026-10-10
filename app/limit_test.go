package app

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	if !limiter.Allow("1.1.1.1") {
		t.Errorf("expected true")
	}
	if !limiter.Allow("1.1.1.1") {
		t.Errorf("expected true")
	}
	if limiter.Allow("1.1.1.1") {
		t.Errorf("expected false, should be rate limited")
	}

	if !limiter.Allow("2.2.2.2") {
		t.Errorf("expected true for different IP")
	}
}
