package main

import (
	"context"
	"sync"
)

type Job[T any] struct {
	val   T
	index int
}

func WorkerPool[T any, R any](ctx context.Context, worker int, items []T, fn func(context.Context, T) (R, error)) ([]R, error) {
	results := make([]R, len(items))
	jobs := make(chan Job[T], len(items))
	var (
		wg         sync.WaitGroup
		once       sync.Once
		firstError error
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // needs to cancel our child's context
	for range worker {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()

			for job := range jobs {
				if ctx.Err() != nil {
					return
				}
				result, err := fn(ctx, job.val)
				if err != nil {
					once.Do(func() {
						firstError = err
						cancel()
					})
					return
				}
				results[job.index] = result
			}
		}()
	}

	for i, item := range items {
		jobs <- Job[T]{val: item, index: i}
	}
	close(jobs)

	wg.Wait()

	if firstError != nil {
		return results, firstError
	}
	return results, ctx.Err()
}
