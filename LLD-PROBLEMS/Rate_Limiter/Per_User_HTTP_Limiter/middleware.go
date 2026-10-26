package main

import (
	"net/http"
	"strconv"
	"sync"
)

type Limiter struct {
	limitMap map[int]*TokenBucket
	mu       *sync.Mutex
}

func NewLimiter() *Limiter {
	return &Limiter{
		mu:       &sync.Mutex{},
		limitMap: make(map[int]*TokenBucket),
	}
}

func (limiter *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawUserID := r.Header.Get("X-User-ID")
		if rawUserID == "" {
			http.Error(w, "X-User-ID header is required", http.StatusBadRequest)
			return
		}
		userId, err := strconv.Atoi(rawUserID)
		if err != nil {
			http.Error(w, "X-User-ID must be an integer", http.StatusBadRequest)
			return
		}
		if !limiter.RateLimit(userId) {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (limiter *Limiter) RateLimit(userId int) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	val, ok := limiter.limitMap[userId]
	if !ok {
		limiter.limitMap[userId] = NewTokenBucket(3, 2)
		val = limiter.limitMap[userId]
	}
	return val.Allow()
}
