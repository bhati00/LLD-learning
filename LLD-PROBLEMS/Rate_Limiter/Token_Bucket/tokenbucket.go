package main

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu         sync.Mutex
	capacity   float64
	rate       float64
	tokens     float64
	lastRefill time.Time
}

func NewTokenBucket(capacity float64, rate float64) *TokenBucket {
	return &TokenBucket{
		mu:         sync.Mutex{},
		capacity:   capacity,
		rate:       rate,
		lastRefill: time.Now(),
		tokens:     capacity,
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// lapsed time
	now := time.Now()
	lapsedTime := now.Sub(tb.lastRefill).Seconds()

	// refill tokens
	tb.tokens = min(tb.capacity, tb.tokens+lapsedTime*tb.rate)
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false

}
