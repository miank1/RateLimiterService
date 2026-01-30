package main

import (
	"fmt"
	"sync"
	"time"
)

// FixedWindowState holds the state for a key in fixed window
type FixedWindowState struct {
	Count     int
	WindowStart time.Time
}

// FixedWindow implements fixed window rate limiting
type FixedWindow struct {
	windowSize time.Duration
	maxRequests int
	states      map[string]FixedWindowState
	mu          sync.RWMutex
}

func NewFixedWindow(windowSize time.Duration, maxRequests int) *FixedWindow {
	return &FixedWindow{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		states:      make(map[string]FixedWindowState),
	}
}

func (fw *FixedWindow) Allow(key string) bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	now := time.Now()
	state, exists := fw.states[key]
	if !exists || now.Sub(state.WindowStart) >= fw.windowSize {
		// New window
		state = FixedWindowState{Count: 1, WindowStart: now}
		fw.states[key] = state
		return true
	}

	if state.Count < fw.maxRequests {
		state.Count++
		fw.states[key] = state
		return true
	}
	return false
}

// SlidingWindowState holds the timestamps for a key
type SlidingWindowState struct {
	Requests []time.Time
}

// SlidingWindow implements sliding window rate limiting
type SlidingWindow struct {
	windowSize  time.Duration
	maxRequests int
	states      map[string]SlidingWindowState
	mu          sync.RWMutex
}

func NewSlidingWindow(windowSize time.Duration, maxRequests int) *SlidingWindow {
	return &SlidingWindow{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		states:      make(map[string]SlidingWindowState),
	}
}

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
		sw.states[key] = state
		return true
	}
	return false
}

func main() {
	// Compare Fixed Window vs Sliding Window
	// Window: 10 seconds, Max: 5 requests

	fw := NewFixedWindow(10*time.Second, 5)
	sw := NewSlidingWindow(10*time.Second, 5)

	key := "user1"

	fmt.Println("=== Fixed Window Test ===")
	// Simulate requests over time
	for i := 0; i < 5; i++ {
		fmt.Printf("Request %d: %t\n", i+1, fw.Allow(key))
	}
	fmt.Printf("Request 6: %t (should be false)\n", fw.Allow(key))

	// Wait just after window start (simulate boundary burst)
	time.Sleep(10 * time.Second)
	fmt.Printf("Request after window: %t\n", fw.Allow(key)) // New window, allows

	fmt.Println("\n=== Sliding Window Test ===")
	// Reset for sliding window
	sw = NewSlidingWindow(10*time.Second, 5)

	for i := 0; i < 5; i++ {
		fmt.Printf("Request %d: %t\n", i+1, sw.Allow(key))
		time.Sleep(1 * time.Second) // Spread out
	}
	fmt.Printf("Request 6: %t (should be false)\n", sw.Allow(key))

	// Wait 6 seconds, some requests slide out
	time.Sleep(6 * time.Second)
	fmt.Printf("Request after slide: %t\n", sw.Allow(key)) // Should allow since window slid
}