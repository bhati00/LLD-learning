# Worker Pool Pattern

> **Process jobs with a fixed number of workers to bound concurrency and resource use.**

A worker pool is useful when many independent jobs share the same processing function. It provides an explicit concurrency limit instead of starting an unbounded goroutine for every job.

---

## The Core Idea

Send jobs through a queue. A fixed number of worker goroutines read from that queue and process jobs. The pool controls parallelism; callers define job behavior and decide how results and errors are collected.

---

## Example: Bounded Job Processing

```go
package workerpool

import (
	"context"
	"sync"
)

type Job struct {
	ID  int
	Run func(context.Context) error
}

type Result struct {
	ID  int
	Err error
}

func Run(ctx context.Context, workers int, jobs []Job) []Result {
	if workers < 1 {
		workers = 1
	}

	queue := make(chan Job)
	results := make(chan Result, len(jobs))
	var group sync.WaitGroup

	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for job := range queue {
				err := ctx.Err()
				if err == nil {
					err = job.Run(ctx)
				}
				results <- Result{ID: job.ID, Err: err}
			}
		}()
	}

	go func() {
		defer close(queue)
		for _, job := range jobs {
			select {
			case queue <- job:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		group.Wait()
		close(results)
	}()

	completed := make([]Result, 0, len(jobs))
	for result := range results {
		completed = append(completed, result)
	}
	return completed
}
```

Results arrive in completion order, not input order. If callers need stable ordering, sort by `ID` or place results into indexed slots. On cancellation, jobs not yet submitted are omitted; submitted jobs report the context error or the error returned by their `Run` function.

---

## Pool Design Decisions

- **Concurrency:** fixed count, often derived from CPU, I/O limits, or downstream capacity.
- **Queue capacity:** bounded buffering provides backpressure; an unbounded queue can exhaust memory.
- **Failure policy:** stop on first error, collect all errors, or retry selected failures.
- **Cancellation:** decide whether queued jobs are dropped and whether running jobs must honor context.
- **Lifecycle:** define who submits jobs, closes the queue, and waits for completion.

## When It Helps

- Work items are independent and use the same processing logic.
- The system must cap concurrent database, network, or CPU work.
- A bounded queue can provide backpressure to producers.

## Costs and Pitfalls

- A pool adds queueing latency and lifecycle complexity.
- Too few workers underuse capacity; too many can overload downstream systems.
- Workers must not outlive their owner or ignore cancellation indefinitely.
- Do not build a custom pool when a simple semaphore or a standard library primitive is enough.
- Avoid sharing mutable job state without synchronization.

## Interview Check

Explain how the worker count is chosen, how the queue is bounded, what happens on cancellation and failure, and who closes the job channel. Compare with Pipeline: workers perform similar jobs; pipeline stages perform different steps in sequence.