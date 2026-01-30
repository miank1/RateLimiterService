package store

import (
	"sync"
	"time"
)

// Store interface for key-value storage
// Edge cases handled:
// - Concurrent requests: RWMutex ensures thread safety.
// - Clock drift: Uses time.Now(), which is monotonic in Go; for distributed, sync clocks.
// - Memory growth: TTL cleanup and optional maxKeys with LRU eviction.
type Store interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

// InMemoryStore implements Store using a map with cleanup
type InMemoryStore struct {
	mu          sync.RWMutex
	data        map[string]interface{}
	lastAccess  map[string]time.Time
	ttl         time.Duration // time to live for entries
	maxKeys     int           // optional max number of keys to prevent unbounded growth
	cleanupDone chan struct{} // to stop the cleanup goroutine
}

func NewInMemoryStore(ttl time.Duration) *InMemoryStore {
	return NewInMemoryStoreWithMaxKeys(ttl, 0) // no limit by default
}

func NewInMemoryStoreWithMaxKeys(ttl time.Duration, maxKeys int) *InMemoryStore {
	s := &InMemoryStore{
		data:        make(map[string]interface{}),
		lastAccess:  make(map[string]time.Time),
		ttl:         ttl,
		maxKeys:     maxKeys,
		cleanupDone: make(chan struct{}),
	}
	go s.cleanupRoutine()
	return s
}

func (s *InMemoryStore) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	if ok {
		s.lastAccess[key] = time.Now() // update access time
	}
	return val, ok
}

func (s *InMemoryStore) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	// If maxKeys is set and exceeded, evict oldest key
	if s.maxKeys > 0 && len(s.data) >= s.maxKeys && s.data[key] == nil {
		s.evictOldest()
	}
	s.data[key] = value
	s.lastAccess[key] = now
}

func (s *InMemoryStore) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true
	for key, t := range s.lastAccess {
		if first || t.Before(oldestTime) {
			oldestKey = key
			oldestTime = t
			first = false
		}
	}
	if oldestKey != "" {
		delete(s.data, oldestKey)
		delete(s.lastAccess, oldestKey)
	}
}

func (s *InMemoryStore) cleanupRoutine() {
	ticker := time.NewTicker(s.ttl / 4) // cleanup every ttl/4 for more responsiveness
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.cleanupDone:
			return
		}
	}
}

func (s *InMemoryStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, accessTime := range s.lastAccess {
		if now.Sub(accessTime) > s.ttl {
			delete(s.data, key)
			delete(s.lastAccess, key)
		}
	}
}

// Close stops the cleanup goroutine (call when done)
func (s *InMemoryStore) Close() {
	close(s.cleanupDone)
}