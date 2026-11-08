package main

import (
	"sync"
	"time"
)

/*
 */

type TokenBucket struct {
	burst       float64
	rate        float64
	current     float64
	lastAccrued time.Time
	mu          sync.Mutex
}

func NewTokenBucket(burst float64, rate float64) *TokenBucket {
	return &TokenBucket{
		burst:       burst,
		rate:        rate,
		current:     burst,
		lastAccrued: time.Now(),
		mu:          sync.Mutex{},
	}
}

// it'll first update the balance based upon the formula = min(burst, current + earned)
// where earned = accural duration * rate
// accural duration = curr time - last accured at
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	refill(tb)
	if tb.current >= 1 {
		tb.current -= 1
		return true
	}
	return false
}

func refill(tb *TokenBucket) {
	now := time.Now()
	elapsed := now.Sub(tb.lastAccrued).Seconds()
	tb.current = min(tb.burst, elapsed*tb.rate+tb.current)
	tb.lastAccrued = now
}
