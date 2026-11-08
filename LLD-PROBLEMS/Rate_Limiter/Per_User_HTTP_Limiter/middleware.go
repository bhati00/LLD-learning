package main

import (
	"net/http"
	"sync"
)

/*
1. create the rate limiter : usage registry
2. define method reciver : allowing an ID to proceed
*/

type RateLimiter struct {
	registry map[string]*TokenBucket
	mu       sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		registry: map[string]*TokenBucket{},
		mu:       sync.Mutex{},
	}
}

// it'll accept the handler function and return the handler function only
// let's we are extracting ID from the and do the map lookup
// if the shit not exists than create one and call the allow method
func (rt *RateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("User-ID")
		if userId == "" {
			http.Error(w, "missing id", http.StatusUnauthorized)
			return
		}
		tb := rt.getBucket(userId)
		if !tb.Allow() {
			http.Error(w, "asdf", http.StatusTooManyRequests)
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *RateLimiter) getBucket(id string) *TokenBucket {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	val, ok := rt.registry[id]
	if ok {
		return val
	}
	tb := NewTokenBucket(3, 2)
	rt.registry[id] = tb
	return tb
}
