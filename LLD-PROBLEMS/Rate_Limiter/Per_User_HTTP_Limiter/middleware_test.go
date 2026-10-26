package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMiddlewareValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		userID  string
		message string
	}{
		{"missing", "", "X-User-ID header is required\n"},
		{"malformed", "abc", "X-User-ID must be an integer\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewLimiter()
			called := false
			handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))
			req := httptest.NewRequest(http.MethodGet, "/hello", nil)
			req.Header.Set("X-User-ID", tc.userID)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest || rec.Body.String() != tc.message {
				t.Fatalf("got %d %q; want 400 %q", rec.Code, rec.Body.String(), tc.message)
			}
			if called || len(limiter.limitMap) != 0 {
				t.Fatal("invalid identity reached the handler or created a bucket")
			}
		})
	}
}

func TestMiddlewareBurstIsolationAndRefill(t *testing.T) {
	limiter := NewLimiter()
	// Disable refill during the burst so its result does not depend on test speed.
	limiter.limitMap[101] = NewTokenBucket(3, 0)
	calls := 0
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte("greeting"))
	}))
	check := func(userID string, wantStatus int, wantBody string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/hello", nil)
		req.Header.Set("X-User-ID", userID)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != wantStatus || rec.Body.String() != wantBody {
			t.Fatalf("user %s: got %d %q; want %d %q",
				userID, rec.Code, rec.Body.String(), wantStatus, wantBody)
		}
	}
	for i := 0; i < 3; i++ {
		check("101", http.StatusOK, "greeting")
	}
	check("101", http.StatusTooManyRequests, "Too many requests\n")
	check("202", http.StatusOK, "greeting")
	if calls != 4 {
		t.Fatalf("handler called %d times; want 4", calls)
	}

	bucket := limiter.limitMap[101]
	bucket.mu.Lock()
	bucket.rate = 2
	bucket.lastRefill = time.Now().Add(-2 * time.Second)
	bucket.mu.Unlock()
	check("101", http.StatusOK, "greeting")
}

func TestMiddlewareConcurrentFirstRequests(t *testing.T) {
	limiter := NewLimiter()
	var calls atomic.Int32
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req := httptest.NewRequest(http.MethodGet, "/hello", nil)
			req.Header.Set("X-User-ID", "303")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK && rec.Code != http.StatusTooManyRequests {
				t.Errorf("unexpected status: %d", rec.Code)
			}
			if rec.Code == http.StatusTooManyRequests && strings.TrimSpace(rec.Body.String()) != "Too many requests" {
				t.Errorf("unexpected rejection body: %q", rec.Body.String())
			}
		}()
	}
	started := time.Now()
	close(start)
	wg.Wait()
	maxAllowed := 3 + int(time.Since(started).Seconds()*2)
	if len(limiter.limitMap) != 1 {
		t.Fatalf("got %d buckets; want 1", len(limiter.limitMap))
	}
	if admitted := int(calls.Load()); admitted < 3 || admitted > maxAllowed {
		t.Fatalf("%d requests admitted; want between 3 and %d", admitted, maxAllowed)
	}
	// Check accounting without assuming the scheduler finished before a refill.
	bucket := limiter.limitMap[303]
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.tokens < 0 || bucket.tokens > bucket.capacity {
		t.Fatalf("tokens outside bucket bounds: %v", bucket.tokens)
	}
}
