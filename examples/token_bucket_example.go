package main

import (
	"fmt"
	"sync"
	"time"
)

// TokenBucketState holds the state for a key
type TokenBucketState struct {
	Tokens   int64
	LastTime time.Time
}

// TokenBucket implements the token bucket algorithm with concurrency safety
type TokenBucket struct {
	capacity int64         // max tokens
	rate     int64         // tokens per second
	states   map[string]TokenBucketState
	mu       sync.RWMutex  // protects the states map
}

func NewTokenBucket(capacity, rate int64) *TokenBucket {
	return &TokenBucket{
		capacity: capacity,
		rate:     rate,
		states:   make(map[string]TokenBucketState),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (tb *TokenBucket) Allow(key string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	state, exists := tb.states[key]
	if !exists {
		// Initialize with full capacity
		state = TokenBucketState{Tokens: tb.capacity, LastTime: now}
		tb.states[key] = state
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(state.LastTime)
	tokensToAdd := elapsed.Nanoseconds() * tb.rate / int64(time.Second)
	state.Tokens += tokensToAdd
	if state.Tokens > tb.capacity {
		state.Tokens = tb.capacity
	}

	if state.Tokens > 0 {
		state.Tokens--
		state.LastTime = now
		tb.states[key] = state
		return true
	}
	return false
}

// GetRemaining returns the remaining tokens for a key
func (tb *TokenBucket) GetRemaining(key string) int64 {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	state, exists := tb.states[key]
	if !exists {
		return tb.capacity
	}

	now := time.Now()
	elapsed := now.Sub(state.LastTime)
	tokensToAdd := elapsed.Nanoseconds() * tb.rate / int64(time.Second)
	remaining := state.Tokens + tokensToAdd
	if remaining > tb.capacity {
		remaining = tb.capacity
	}
	return remaining
}

func main() {
	// Example usage: 10 tokens capacity, 1 token per second
	tb := NewTokenBucket(10, 1)

	key := "user1"

	// Allow 10 requests immediately
	for i := 0; i < 10; i++ {
		if tb.Allow(key) {
			fmt.Printf("Request %d allowed, remaining: %d\n", i+1, tb.GetRemaining(key))
		}
	}

	// Next should be denied
	if tb.Allow(key) {
		fmt.Println("Unexpected: request allowed")
	} else {
		fmt.Println("Request denied as expected")
	}

	// Wait for refill
	time.Sleep(11 * time.Second)

	// Now allow again
	if tb.Allow(key) {
		fmt.Println("Request allowed after refill")
	}
}