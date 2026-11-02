package main

import (
	"context"
	"fmt"
	"sync"
)

/*
!SECTION
FLOW : we run n workers that continously consume the jobs channel, now for the prepartion of jobs
channel depends upon the buffer of the jobs channel, if we have can preapre any where if not than
first run the workers (helps in maintainng the backpressue)
talking about the context signal, we need to derive the child's context from parent
and use the cancel signal
*
*/
type Job[T any] struct {
	task  T
	index int
}

func WorkerPool[T, R any](ctx context.Context, n int, items []T, fn func(context.Context, T) (R, error)) ([]R, error) {
	if n < 1 {
		n = 1
	}
	results := make([]R, len(items))
	jobs := make(chan Job[T])
	var firstError error
	var once sync.Once
	var wg sync.WaitGroup
	childctx, cancel := context.WithCancel(ctx) // creating the child context
	defer cancel()                              // frees childctx on return; cancellation only flows parent to child

	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if childctx.Err() != nil {
					return
				}
				result, err := func() (r R, err error) {
					defer func() {
						if p := recover(); p != nil {
							err = fmt.Errorf("panic in task %d: %v", job.index, p)
						}
					}()
					return fn(childctx, job.task)
				}()
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

produce:
	for idx, val := range items {
		select {
		case jobs <- Job[T]{task: val, index: idx}:
		case <-childctx.Done():
			break produce // a bare break would only exit the select
		}
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
