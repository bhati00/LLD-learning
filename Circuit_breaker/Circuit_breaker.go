package main

import (
	"sync"
	"time"
)

type Status int

const (
	Closed Status = iota
	Open
	HalfOpen
)

type Config struct {
	failureThreshold int
	halfOpenTimeout  time.Duration // how long to stay open before moving to half-open
	maxProbes        int
}

type CircuitBreaker struct {
	mu             sync.Mutex
	current        Status
	failCounter    int
	config         Config
	openedAt       time.Time
	probesLeft     int
	probeSuccesses int
}

func NewCircuitBreaker(failureThreshold int, halfOpenTimeout time.Duration, maxProbes int) *CircuitBreaker {
	return &CircuitBreaker{
		config: Config{
			failureThreshold: failureThreshold,
			halfOpenTimeout:  halfOpenTimeout,
			maxProbes:        maxProbes,
		},
		current: Closed,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.current == Open && time.Since(cb.openedAt) >= cb.config.halfOpenTimeout {
		cb.current = HalfOpen
		cb.probesLeft = cb.config.maxProbes
		cb.probeSuccesses = 0
	}
	switch cb.current {
	case Closed:
		return true
	case HalfOpen:
		if cb.probesLeft > 0 {
			cb.probesLeft--
			return true
		}
	}
	return false
}

func (cb *CircuitBreaker) State() Status {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.current
}

func (cb *CircuitBreaker) Record(failed bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.current {
	case Closed:
		if !failed {
			cb.failCounter = 0
			return
		}
		cb.failCounter++
		if cb.failCounter >= cb.config.failureThreshold {
			cb.trip()
		}
	case HalfOpen:
		if failed {
			cb.trip()
			return
		}
		cb.probeSuccesses++
		if cb.probeSuccesses >= cb.config.maxProbes {
			cb.current = Closed
			cb.failCounter = 0
		}
	}
}

// trip must be called with cb.mu held.
func (cb *CircuitBreaker) trip() {
	cb.current = Open
	cb.openedAt = time.Now()
	cb.failCounter = 0
}
