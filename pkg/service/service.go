package service

import (
	"time"

	"RateLimiterService/pkg/clock"
	"RateLimiterService/pkg/ratelimiter"
	"RateLimiterService/pkg/store"
)

// Config holds the configuration for the rate limiter service
type Config struct {
	Algorithm        string
	Capacity         int64
	Rate             int64
	WindowSize       time.Duration
	MaxRequests      int
	TTL              time.Duration
	MaxKeys          int // max keys in store to prevent memory growth
}

// Decision represents the result of a rate limit check
type Decision struct {
	Allowed   bool
	Remaining int64
}

// RateLimitService encapsulates the rate limiting logic
type RateLimitService struct {
	limiter ratelimiter.RateLimiter
}

// NewRateLimitService creates a new service based on config
func NewRateLimitService(config Config) *RateLimitService {
	c := clock.RealClock{}
	s := store.NewInMemoryStoreWithMaxKeys(config.TTL, config.MaxKeys)

	var limiter ratelimiter.RateLimiter
	switch config.Algorithm {
	case "tokenbucket":
		limiter = ratelimiter.NewTokenBucket(config.Capacity, config.Rate, c, s)
	case "slidingwindow":
		limiter = ratelimiter.NewSlidingWindow(config.WindowSize, config.MaxRequests, c, s)
	default:
		// Default to token bucket
		limiter = ratelimiter.NewTokenBucket(10, 1, c, s)
	}

	return &RateLimitService{limiter: limiter}
}

// CheckRateLimit checks if a request is allowed for the given key
func (s *RateLimitService) CheckRateLimit(key string) Decision {
	allowed, remaining := s.limiter.Allow(key)
	return Decision{Allowed: allowed, Remaining: remaining}
}