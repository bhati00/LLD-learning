package main

import (
	"sync"
	"time"
)

type FixedWindow struct {
	mu             sync.Mutex
	capacity       int
	windowDuration time.Duration
	windowStart    time.Time
	balance        int
}

func NewFixedWindow(capacity int, windowDuration time.Duration) *FixedWindow {
	return &FixedWindow{
		capacity:       capacity,
		windowDuration: windowDuration,
		windowStart:    time.Now().Truncate(windowDuration),
		balance:        capacity,
	}
}

func (fw *FixedWindow) Admit() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	now := time.Now()
	windowEnd := fw.windowStart.Add(fw.windowDuration)
	if !now.Before(windowEnd) {
		fw.balance = fw.capacity
		fw.windowStart = now.Truncate(fw.windowDuration)
	}

	if fw.balance <= 0 {
		return false
	}

	fw.balance--
	return true
}
