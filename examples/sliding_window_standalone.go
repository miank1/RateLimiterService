package main

import (
	"fmt"
	"sync"
	"time"
)

// SlidingWindowState holds the timestamps for a key
type SlidingWindowState struct {
	Requests []time.Time
}

// SlidingWindow implements the sliding window rate limiting algorithm with concurrency safety
type SlidingWindow struct {
	windowSize  time.Duration // size of the sliding window
	maxRequests int           // max requests per window
	states      map[string]SlidingWindowState
	mu          sync.RWMutex  // protects the states map
}

func NewSlidingWindow(windowSize time.Duration, maxRequests int) *SlidingWindow {
	return &SlidingWindow{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		states:      make(map[string]SlidingWindowState),
	}
}

// Allow checks if a request is allowed based on the sliding window
func (sw *SlidingWindow) Allow(key string) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.windowSize)

	state, exists := sw.states[key]
	if !exists {
		state = SlidingWindowState{Requests: []time.Time{now}}
		sw.states[key] = state
		return true
	}

	// Filter out requests outside the window
	validReqs := []time.Time{}
	for _, t := range state.Requests {
		if t.After(windowStart) {
			validReqs = append(validReqs, t)
		}
	}

	if len(validReqs) < sw.maxRequests {
		validReqs = append(validReqs, now)
		state.Requests = validReqs
		sw.states[key] = state
		return true
	}
	return false
}

// GetRemaining returns the number of remaining requests in the current window
func (sw *SlidingWindow) GetRemaining(key string) int {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-sw.windowSize)

	state, exists := sw.states[key]
	if !exists {
		return sw.maxRequests
	}

	validCount := 0
	for _, t := range state.Requests {
		if t.After(windowStart) {
			validCount++
		}
	}
	return sw.maxRequests - validCount
}

func main() {
	// Example: 10 requests per 60 seconds
	sw := NewSlidingWindow(60*time.Second, 10)

	key := "user1"

	// Allow 10 requests
	for i := 0; i < 10; i++ {
		if sw.Allow(key) {
			fmt.Printf("Request %d allowed, remaining: %d\n", i+1, sw.GetRemaining(key))
		}
		time.Sleep(1 * time.Second) // Simulate time passing
	}

	// 11th should be denied
	if sw.Allow(key) {
		fmt.Println("Unexpected: request allowed")
	} else {
		fmt.Println("Request 11 denied as expected")
	}

	// Wait for some requests to slide out (after 11 seconds, first request expires)
	time.Sleep(11 * time.Second)

	// Now allow again
	if sw.Allow(key) {
		fmt.Printf("Request allowed after slide, remaining: %d\n", sw.GetRemaining(key))
	}
}