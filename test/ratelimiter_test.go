package ratelimiter

import (
	"testing"
	"time"

	"RateLimiterService/pkg/clock"
	"RateLimiterService/pkg/store"
)

func TestTokenBucket(t *testing.T) {
	c := clock.RealClock{}
	s := store.NewInMemoryStore(1 * time.Hour)
	tb := NewTokenBucket(5, 1, c, s)
	key := "test"

	// Should allow 5 requests immediately
	for i := 0; i < 5; i++ {
		allowed, _ := tb.Allow(key)
		if !allowed {
			t.Errorf("Expected allow, got deny at request %d", i+1)
		}
	}

	// 6th should deny
	allowed, _ := tb.Allow(key)
	if allowed {
		t.Error("Expected deny, got allow")
	}

	// Wait for refill
	time.Sleep(6 * time.Second)

	// Should allow again
	allowed, _ := tb.Allow(key)
	if !allowed {
		t.Error("Expected allow after refill")
	}
}

func TestSlidingWindow(t *testing.T) {
	c := clock.RealClock{}
	s := store.NewInMemoryStore(1 * time.Hour)
	sw := NewSlidingWindow(10*time.Second, 3, c, s)
	key := "test"

	// Allow 3 requests
	for i := 0; i < 3; i++ {
		allowed, _ := sw.Allow(key)
		if !allowed {
			t.Errorf("Expected allow, got deny at request %d", i+1)
		}
	}

	// 4th should deny
	allowed, _ := sw.Allow(key)
	if allowed {
		t.Error("Expected deny, got allow")
	}

	// Wait for window to slide
	time.Sleep(11 * time.Second)

	// Should allow again
	allowed, _ := sw.Allow(key)
	if !allowed {
		t.Error("Expected allow after window slides")
	}
}