package service

import (
	"testing"
	"time"
)

func TestRateLimitService_TokenBucket(t *testing.T) {
	config := Config{
		Algorithm: "tokenbucket",
		Capacity:  5,
		Rate:      1,
		TTL:       1 * time.Hour,
	}
	svc := NewRateLimitService(config)

	key := "test"

	// Allow 5 requests
	for i := 0; i < 5; i++ {
		decision := svc.CheckRateLimit(key)
		if !decision.Allowed {
			t.Errorf("Expected allow at %d", i)
		}
	}

	// Deny 6th
	decision := svc.CheckRateLimit(key)
	if decision.Allowed {
		t.Error("Expected deny")
	}
}

func TestRateLimitService_SlidingWindow(t *testing.T) {
	config := Config{
		Algorithm:   "slidingwindow",
		WindowSize:  10 * time.Second,
		MaxRequests: 3,
		TTL:         1 * time.Hour,
	}
	svc := NewRateLimitService(config)

	key := "test"

	// Allow 3 requests
	for i := 0; i < 3; i++ {
		decision := svc.CheckRateLimit(key)
		if !decision.Allowed {
			t.Errorf("Expected allow at %d", i)
		}
		time.Sleep(1 * time.Second)
	}

	// Deny 4th
	decision := svc.CheckRateLimit(key)
	if decision.Allowed {
		t.Error("Expected deny")
	}
}