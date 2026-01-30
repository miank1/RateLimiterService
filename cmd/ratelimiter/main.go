package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"RateLimiterService/pkg/service"
)

type CheckRequest struct {
	Key string `json:"key"`
}

type CheckResponse struct {
	Allowed   bool   `json:"allowed"`
	Remaining int64  `json:"remaining,omitempty"`
	ResetAt   string `json:"reset_at,omitempty"`
}

func main() {
	algorithm := os.Getenv("ALGORITHM")
	if algorithm == "" {
		algorithm = "tokenbucket"
	}

	// Parse config from env
	ttlStr := os.Getenv("TTL_SECONDS")
	ttlSec, _ := strconv.Atoi(ttlStr)
	if ttlSec == 0 {
		ttlSec = 3600
	}
	ttl := time.Duration(ttlSec) * time.Second

	maxKeysStr := os.Getenv("MAX_KEYS")
	maxKeys, _ := strconv.Atoi(maxKeysStr)
	// default 0 (unlimited)

	config := service.Config{
		Algorithm: algorithm,
		TTL:       ttl,
		MaxKeys:   maxKeys,
	}

	switch algorithm {
	case "tokenbucket":
		capacityStr := os.Getenv("CAPACITY")
		capacity, _ := strconv.ParseInt(capacityStr, 10, 64)
		if capacity == 0 {
			capacity = 10
		}
		rateStr := os.Getenv("RATE")
		rate, _ := strconv.ParseInt(rateStr, 10, 64)
		if rate == 0 {
			rate = 1
		}
		config.Capacity = capacity
		config.Rate = rate
	case "slidingwindow":
		windowSizeStr := os.Getenv("WINDOW_SIZE_SECONDS")
		windowSizeSec, _ := strconv.Atoi(windowSizeStr)
		if windowSizeSec == 0 {
			windowSizeSec = 60
		}
		maxRequestsStr := os.Getenv("MAX_REQUESTS")
		maxRequests, _ := strconv.Atoi(maxRequestsStr)
		if maxRequests == 0 {
			maxRequests = 10
		}
		config.WindowSize = time.Duration(windowSizeSec) * time.Second
		config.MaxRequests = maxRequests
	default:
		fmt.Println("Invalid algorithm")
		os.Exit(1)
	}

	svc := service.NewRateLimitService(config)

	http.HandleFunc("/api/v1/rate-limit/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		key := req.Key
		if key == "" {
			key = r.RemoteAddr
		}

		decision := svc.CheckRateLimit(key)
		resp := CheckResponse{Allowed: decision.Allowed, Remaining: decision.Remaining}
		if decision.Allowed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusTooManyRequests)
			// For simplicity, no reset_at
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Starting server on port %s with %s\n", port, algorithm)
	http.ListenAndServe(":"+port, nil)
}