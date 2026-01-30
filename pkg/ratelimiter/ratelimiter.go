package ratelimiter

import (
	"time"

	"RateLimiterService/pkg/clock"
	"RateLimiterService/pkg/store"
)

// Edge cases handled:
// - Clock drift: Algorithms use elapsed time calculations, resilient to small drifts.
// - Concurrent: Store handles locking; algorithms are stateless per call.
// - Memory: Per-key state is managed by Store; SlidingWindow filters old timestamps.

// RateLimiter interface for different rate limiting algorithms
type RateLimiter interface {
	Allow(key string) (bool, int64)
}

// TokenBucketState holds the state for a key
type TokenBucketState struct {
	Tokens   int64
	LastTime time.Time
}

// TokenBucket implementation
type TokenBucket struct {
	capacity int64
	rate     int64
	clock    clock.Clock
	store    store.Store
}

func NewTokenBucket(capacity, rate int64, clock clock.Clock, store store.Store) *TokenBucket {
	return &TokenBucket{
		capacity: capacity,
		rate:     rate,
		clock:    clock,
		store:    store,
	}
}

func (tb *TokenBucket) Allow(key string) (bool, int64) {
	now := tb.clock.Now()

	val, exists := tb.store.Get(key)
	var state TokenBucketState
	if !exists {
		state = TokenBucketState{Tokens: tb.capacity, LastTime: now}
		tb.store.Set(key, state)
		return true, tb.capacity - 1
	}
	state = val.(TokenBucketState)

	elapsed := now.Sub(state.LastTime)
	tokensToAdd := elapsed.Nanoseconds() * tb.rate / int64(time.Second)
	state.Tokens += tokensToAdd
	if state.Tokens > tb.capacity {
		state.Tokens = tb.capacity
	}

	if state.Tokens > 0 {
		state.Tokens--
		state.LastTime = now
		tb.store.Set(key, state)
		return true, state.Tokens
	}
	return false, 0
}

// SlidingWindowState holds the timestamps for a key
type SlidingWindowState struct {
	Requests []time.Time
}

// SlidingWindow implementation
type SlidingWindow struct {
	windowSize  time.Duration
	maxRequests int
	clock       clock.Clock
	store       store.Store
}

func NewSlidingWindow(windowSize time.Duration, maxRequests int, clock clock.Clock, store store.Store) *SlidingWindow {
	return &SlidingWindow{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		clock:       clock,
		store:       store,
	}
}

func (sw *SlidingWindow) Allow(key string) (bool, int64) {
	now := sw.clock.Now()
	windowStart := now.Add(-sw.windowSize)

	val, exists := sw.store.Get(key)
	var state SlidingWindowState
	if !exists {
		state = SlidingWindowState{Requests: []time.Time{}}
	} else {
		state = val.(SlidingWindowState)
	}

	// Remove old requests
	validReqs := []time.Time{}
	for _, t := range state.Requests {
		if t.After(windowStart) {
			validReqs = append(validReqs, t)
		}
	}

	if len(validReqs) < sw.maxRequests {
		validReqs = append(validReqs, now)
		state.Requests = validReqs
		sw.store.Set(key, state)
		return true, int64(sw.maxRequests - len(validReqs))
	}
	return false, 0
}