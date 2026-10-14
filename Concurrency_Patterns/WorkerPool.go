package main

import (
	"context"
	"errors"
	"sync"
)

type job[T any] struct {
	val T
	idx int
}

func WorkerPool[T, R any](ctx context.Context, inputs []T, task func(context.Context, T) (R, error), workers int) ([]R, error) {
	if workers <= 0 {
		return nil, errors.New("workers must be positive")
	}
	jobs := make(chan job[T], len(inputs))
	results := make([]R, len(inputs))
	var once sync.Once
	var wg sync.WaitGroup
	var firstError error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if ctx.Err() != nil {
					return
				}
				result, err := task(ctx, j.val)
				if err != nil {
					once.Do(func() {
						firstError = err
						cancel()
					})
					return
				}
				results[j.idx] = result
			}
		}()
	}

	for idx, val := range inputs {
		jobs <- job[T]{val: val, idx: idx}
	}
	close(jobs)
	wg.Wait()
	if firstError != nil {
		return nil, firstError
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
