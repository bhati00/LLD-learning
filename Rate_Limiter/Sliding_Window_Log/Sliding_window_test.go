package main

import (
	"sync"
	"testing"
	"time"
)

func TestRemoveExpired(t *testing.T) {
	now := time.Date(2026, time.October, 23, 12, 0, 20, 0, time.UTC)
	cutoff := now.Add(-10 * time.Second)
	tests := []struct {
		name    string
		history []time.Time
		want    []time.Time
	}{
		{name: "empty"},
		{
			name:    "none expired",
			history: []time.Time{cutoff.Add(time.Nanosecond), now},
			want:    []time.Time{cutoff.Add(time.Nanosecond), now},
		},
		{
			name:    "expired prefix including exact cutoff",
			history: []time.Time{cutoff.Add(-time.Second), cutoff, now},
			want:    []time.Time{now},
		},
		{
			name:    "all expired",
			history: []time.Time{cutoff.Add(-time.Second), cutoff},
		},
		{
			name:    "duplicate active timestamps",
			history: []time.Time{now, now},
			want:    []time.Time{now, now},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw := NewSlidingWindow(3, 10*time.Second)
			sw.window = tt.history
			sw.mu.Lock()
			sw.removeExpired(now)
			sw.mu.Unlock()

			if len(sw.window) != len(tt.want) {
				t.Fatalf("remaining count = %d, want %d", len(sw.window), len(tt.want))
			}
			for i := range tt.want {
				if !sw.window[i].Equal(tt.want[i]) {
					t.Fatalf("entry %d = %v, want %v", i, sw.window[i], tt.want[i])
				}
			}
		})
	}
}

func TestSlidingWindowAdmission(t *testing.T) {
	sw := NewSlidingWindow(2, time.Hour)
	if !sw.Admit() || !sw.Admit() {
		t.Fatal("initial allowance was rejected")
	}
	if sw.Admit() {
		t.Fatal("request beyond capacity was admitted")
	}
	if len(sw.window) != 2 {
		t.Fatalf("history length = %d, want 2", len(sw.window))
	}
}

func TestSlidingWindowAdmissionExpiresOldEntries(t *testing.T) {
	sw := NewSlidingWindow(2, time.Hour)
	old := time.Now().Add(-2 * time.Hour)
	sw.window = []time.Time{old, old}

	if !sw.Admit() {
		t.Fatal("request after expiration was rejected")
	}
	if len(sw.window) != 1 {
		t.Fatalf("history length = %d, want 1", len(sw.window))
	}
}

func TestSlidingWindowConcurrentAdmission(t *testing.T) {
	sw := NewSlidingWindow(3, time.Hour)
	results := make(chan bool, 20)
	var wg sync.WaitGroup
	for i := 0; i < cap(results); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- sw.Admit()
		}()
	}
	wg.Wait()
	close(results)

	allowed := 0
	for result := range results {
		if result {
			allowed++
		}
	}
	if allowed != 3 {
		t.Fatalf("admitted %d requests, want 3", allowed)
	}
}
