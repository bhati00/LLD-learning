package main

import (
	"context"
	"sync"
)

func SemaphorePattern[T any, R any](ctx context.Context, n int, items []T, fn func(context.Context, T) (R, error)) ([]R, error) {

	sema := make(chan struct{}, n)
	output := make([]R, len(items))
	var firstErr error
	ctx, cancel := context.WithCancel(ctx)
	var once sync.Once
	defer cancel()

	var wg sync.WaitGroup
loop:
	for i, val := range items {
		// select picks randomly when both cases are ready, so check first.
		if ctx.Err() != nil {
			break loop
		}
		select {
		case sema <- struct{}{}:
		case <-ctx.Done():
			break loop
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sema }()

			result, err := fn(ctx, val)
			if err != nil {
				once.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}
			output[i] = result
		}()
	}
	wg.Wait()

	// firstErr is the root cause; ctx.Err() covers parent cancellation with no worker error.
	if firstErr != nil {
		return nil, firstErr
	}
	return output, ctx.Err()
}
