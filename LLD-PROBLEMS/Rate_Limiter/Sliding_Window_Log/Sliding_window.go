package main

import (
	"sync"
	"time"
)

type SlidingWindow struct {
	capacity       int
	mu             sync.Mutex
	windowDuration time.Duration
	window         []time.Time
}

func NewSlidingWindow(capacity int, duration time.Duration) *SlidingWindow {
	return &SlidingWindow{
		capacity:       capacity,
		windowDuration: duration,
		window:         make([]time.Time, 0),
		mu:             sync.Mutex{},
	}
}

// removeExpired requires sw.mu to be held by the caller.
func (sw *SlidingWindow) removeExpired(now time.Time) {
	cutoff := now.Add(-sw.windowDuration)
	expired := 0
	for expired < len(sw.window) && !sw.window[expired].After(cutoff) {
		expired++
	}
	clear(sw.window[:expired])
	sw.window = sw.window[expired:]
}

func (sw *SlidingWindow) Admit() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	sw.removeExpired(now)
	if len(sw.window) < sw.capacity {
		sw.window = append(sw.window, now)
		return true
	}
	return false
}
