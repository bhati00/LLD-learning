# Bounded Parallel Processor

> **Run many independent tasks concurrently, but never more than N at a time, and still get correct results, errors, cancellation, and shutdown.**

All code below was compiled, vetted, and run with `go test -race -count=5` (Go 1.26). `errgroup` needs `golang.org/x/sync`.

Related: [Worker Pool pattern](../DESIGN-PATTERNS/10-Worker-Pool.md) (the pattern); this note is the full exercise.

---

## 1. Problem Statement

Given `items []T` and `fn(ctx, T) (R, error)`, produce `[]R` while:

1. at most `N` calls to `fn` run at the same time
2. results line up with input order
3. the first error cancels remaining work (fail-fast), unless a different policy is requested
4. `ctx` cancellation stops submission and tells running tasks to stop
5. no goroutine outlives the call (no leaks), and a panic in `fn` does not crash the process

### Why bound?

Unbounded `go fn(item)` for 1M items exhausts one of: memory (goroutine stacks), file descriptors, DB connections, or the downstream's rate/capacity. The bound is a **resource-protection contract**, not a speed optimization.

### Choosing N

| Workload | Start with | Reason |
|---|---|---|
| CPU-bound | `runtime.NumCPU()` | More goroutines than cores only adds scheduling overhead |
| I/O-bound (HTTP, DB) | Downstream limit (pool size, rate limit), then tune by measurement | The scarce resource is the remote side, not your CPU |
| Mixed | Separate limits per resource | One number cannot protect two bottlenecks |

Rule: N comes from the scarce resource. Say that sentence in the interview.

---

## 2. Three Ways to Bound (compare these)

| | Semaphore | Worker pool | `errgroup.SetLimit` |
|---|---|---|---|
| Goroutines | 1 per task, at most N alive | N long-lived | 1 per task, at most N alive |
| Mechanism | `chan struct{}` with cap N | Jobs channel + N readers | Built-in |
| Per-worker state / reusable resource | No | Yes | No |
| Code size | Small | Largest | Smallest |
| Cancel on first error | Hand-written | Hand-written | `WithContext` |
| Panic recovery | Hand-written | Hand-written | None |
| Use when | Independent tasks, default choice | Workers own state (conn, buffer) or tasks are tiny | You already depend on `x/sync` |

---

## 3. Implementation A: Semaphore (default choice)

```go
package bounded

import (
	"context"
	"fmt"
	"sync"
)

func MapSem[T, R any](ctx context.Context, items []T, workers int, fn func(context.Context, T) (R, error)) ([]R, error) {
	if workers < 1 {
		workers = 1
	}
	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]R, len(items))
	sem := make(chan struct{}, workers)
	var (
		wg       sync.WaitGroup
		once     sync.Once
		firstErr error
	)
	fail := func(err error) {
		once.Do(func() {
			firstErr = err
			cancel()
		})
	}

loop:
	for i, item := range items {
		select {
		case sem <- struct{}{}: // acquire; blocks when N are in flight
		case <-ctx.Done():
			break loop
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release
			defer func() {
				if r := recover(); r != nil {
					fail(fmt.Errorf("task %d panicked: %v", i, r))
				}
			}()
			v, err := fn(ctx, item)
			if err != nil {
				fail(err)
				return
			}
			results[i] = v
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
```

Why each piece exists:

- **`sem` capacity N** is the bound. Acquire happens in the *submitter loop*, so goroutines are never created beyond N.
- **`results[i] = v`** is race-free: each goroutine writes a distinct slice element, and `wg.Wait()` establishes happens-before for the reader.
- **`sync.Once`** records only the first error and cancels exactly once.
- **`parent` vs derived `ctx`**: after fail-fast, the derived ctx is cancelled by us, so the caller's cancellation must be checked on `parent`.
- **Order of defers** (LIFO): recover runs first, then release, then `Done`. The slot is always released, even on panic.
- **Go 1.22+** gives each loop iteration its own `i`/`item`, so no `i := i` copy is needed.
- Small caveat: if `sem` and `ctx.Done()` are both ready, `select` picks randomly, so one extra task may start after cancel. `fn` must honor `ctx`.

---

## 4. Implementation B: Worker Pool

```go
type job[T any] struct {
	idx  int
	item T
}

func MapPool[T, R any](ctx context.Context, items []T, workers int, fn func(context.Context, T) (R, error)) ([]R, error) {
	if workers < 1 {
		workers = 1
	}
	if workers > len(items) {
		workers = len(items)
	}
	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]R, len(items))
	jobs := make(chan job[T]) // unbuffered: submit blocks until a worker is free
	var (
		wg       sync.WaitGroup
		once     sync.Once
		firstErr error
	)
	fail := func(err error) {
		once.Do(func() {
			firstErr = err
			cancel()
		})
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if ctx.Err() != nil {
					continue // drain so the producer never blocks
				}
				v, err := safeCall(ctx, fn, j.item)
				if err != nil {
					fail(err)
					continue
				}
				results[j.idx] = v
			}
		}()
	}

	for i, item := range items {
		select {
		case jobs <- job[T]{i, item}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(jobs) // only the sender closes
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func safeCall[T, R any](ctx context.Context, fn func(context.Context, T) (R, error), item T) (r R, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return fn(ctx, item)
}
```

Key points:

- **The index travels with the job** (`idx`), so output order is preserved even though completion order is random.
- **Close ownership:** the producer closes `jobs`; workers never close it. Closing from a receiver or from two places panics.
- **Workers keep draining after cancel.** If they returned early, the producer could block forever on `jobs <-` (goroutine leak). Here the producer also selects on `ctx.Done()`, so both sides are safe.
- **Queue capacity = backpressure.** Unbuffered means the producer runs in lockstep with workers. A buffer of `k` lets the producer run `k` ahead. Bound it; never use an unbounded slice queue.

---

## 5. Implementation C: `errgroup`

```go
import "golang.org/x/sync/errgroup"

func MapGroup[T, R any](ctx context.Context, items []T, workers int, fn func(context.Context, T) (R, error)) ([]R, error) {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(max(workers, 1))

	results := make([]R, len(items))
	for i, item := range items {
		if gctx.Err() != nil {
			break // g.Go keeps launching after cancel unless we stop
		}
		g.Go(func() error {
			v, err := fn(gctx, item)
			if err != nil {
				return err
			}
			results[i] = v
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
```

Gotchas found while testing this (my first version failed both tests):

1. **`g.Go` does not stop submitting after an error.** `SetLimit` makes `g.Go` block when full, but it does not skip later calls once the context is cancelled. Without the `gctx.Err()` check, all 1000 tasks still started.
2. **A cancelled parent ctx returns `nil` from `g.Wait()`** if no task returned an error. Check `ctx.Err()` yourself.
3. **`errgroup` does not recover panics.** A panic in a task crashes the process.
4. `g.TryGo` is the non-blocking variant (returns false when the limit is reached). Useful for "reject if busy".

---

## 6. Error Policies

| Policy | How | Use when |
|---|---|---|
| Fail-fast | First error cancels ctx, return it (A, B, C above) | Tasks are all-or-nothing, e.g. building one response |
| Collect all | Per-index `errs[i]`, then `errors.Join(errs...)` | Batch jobs where partial success is valid |
| Partial results | Return `results` plus a per-item status | Caller wants to retry only failures |

Collect-all variant (no cancel on error):

```go
func MapAll[T, R any](ctx context.Context, items []T, workers int, fn func(context.Context, T) (R, error)) ([]R, error) {
	results := make([]R, len(items))
	errs := make([]error, len(items))
	sem := make(chan struct{}, max(workers, 1))
	var wg sync.WaitGroup
	for i, item := range items {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			results[i], errs[i] = fn(ctx, item)
		}()
	}
	wg.Wait()
	return results, errors.Join(errs...)
}
```

Fail-fast returns the *first observed* error, which is not necessarily the lowest index. State this if asked about determinism.

---

## 7. Extensions That Keep the Bound

Wrap `fn`. Because the wrapper runs inside the worker, the concurrency limit still applies to retries and waiting.

**Retry with exponential backoff:**

```go
func Retry[T, R any](attempts int, base time.Duration, fn func(context.Context, T) (R, error)) func(context.Context, T) (R, error) {
	return func(ctx context.Context, item T) (R, error) {
		var zero R
		var err error
		for a := 0; a < attempts; a++ {
			var v R
			if v, err = fn(ctx, item); err == nil {
				return v, nil
			}
			select {
			case <-time.After(base << a):
			case <-ctx.Done():
				return zero, ctx.Err()
			}
		}
		return zero, err
	}
}
```

Production notes: retry only retryable errors (not validation errors), add jitter, cap the total delay. A sleeping retry still holds a worker slot; that is usually desired because it throttles pressure on a struggling downstream.

**Rate limit (starts per second) on top of the concurrency cap:**

```go
func Paced[T, R any](every time.Duration, fn func(context.Context, T) (R, error)) func(context.Context, T) (R, error) {
	tick := time.NewTicker(every)
	return func(ctx context.Context, item T) (R, error) {
		select {
		case <-tick.C:
			return fn(ctx, item)
		case <-ctx.Done():
			var zero R
			return zero, ctx.Err()
		}
	}
}
```

Concurrency limit and rate limit are different knobs: concurrency caps *simultaneous* calls; rate caps *starts per time*. Slow calls hit the first; fast calls hit the second. In production prefer `golang.org/x/time/rate` (token bucket, supports bursts) and stop the ticker when finished.

**Other extensions (describe the approach, not full code):**

- **Per-task timeout:** `ctx, cancel := context.WithTimeout(ctx, d)` inside the wrapper; `defer cancel()`.
- **Weighted tasks:** `semaphore.NewWeighted(total)` with `Acquire(ctx, weight)`; a heavy task takes more permits.
- **Streaming (millions of items):** accept `<-chan T`, return `<-chan Result`; the producer never materializes the slice. Order is then by completion unless you carry the index.
- **Dynamic N:** replace the fixed channel capacity with a resizable semaphore (mutex + cond, or adjust permits on `semaphore.Weighted`); mention adaptive concurrency (AIMD) driven by latency or error rate.
- **Metrics:** gauges for in-flight and queued, histograms for task latency, counters for errors and panics.
- **Graceful shutdown (long-lived pool type):** `Submit` returns `ErrClosed` after `Close()`; `Close` stops intake, drains the queue, then waits on the `WaitGroup`. Add `Shutdown(ctx)` for a deadline.

---

## 8. Backpressure Choices (queue full)

| Choice | Behavior | Trade-off |
|---|---|---|
| Block the producer | `jobs <- j` waits | Natural slow-down; producer must tolerate blocking |
| Reject | Non-blocking send, return `ErrQueueFull` | Caller decides (retry, 429); good for request paths |
| Drop oldest/newest | Discard | Only for lossy data (metrics, telemetry) |
| Unbounded queue | Never blocks | Hides overload until OOM; avoid |

---

## 9. Testing Checklist

Prove the bound with an atomic peak counter, not with sleeps and hope:

```go
var inFlight, peak atomic.Int32
fn := func(ctx context.Context, x int) (int, error) {
	n := inFlight.Add(1)
	defer inFlight.Add(-1)
	for {
		p := peak.Load()
		if n <= p || peak.CompareAndSwap(p, n) {
			break
		}
	}
	time.Sleep(2 * time.Millisecond) // widen the overlap window
	return x * 2, nil
}
// run with N=4 over 50 items; assert peak.Load() <= 4 and out[i] == i*2
```

| Test | Assertion |
|---|---|
| Bound | `peak <= N` |
| Order | `out[i]` corresponds to `items[i]` |
| Fail-fast | Returns the error; far fewer than `len(items)` tasks started |
| Cancel | Returns `context.Canceled`; goroutine count returns to baseline (compare `runtime.NumGoroutine()` before/after, with a short settle delay) |
| Panic | Returns an error, process survives, slot released |
| Edges | empty input, `workers=0`, `workers > len(items)`, already-cancelled ctx |
| Race | `go test -race -count=5` |

Note: a "peak <= N" test cannot prove correctness alone (a bug may simply not overlap on a given run). The structure (acquire before spawn) is what guarantees it; the test guards regressions. For goroutine-leak checks in a real project, `go.uber.org/goleak` is more robust than `NumGoroutine`.

---

## 10. Common Bugs (spot-the-bug material)

1. **Spawn first, acquire inside the goroutine.** All goroutines get created; only execution is bounded. Memory still blows up. Acquire in the loop.
2. **`wg.Add(1)` inside the goroutine.** `Wait` may run before `Add`. Call `Add` before `go`.
3. **Producer blocked forever** after workers exit early. Always `select` on `ctx.Done()` when sending, or keep draining.
4. **Close of a channel from the receiver side** or from multiple goroutines: panic.
5. **Appending to a shared slice from workers** without a lock: data race. Use indexed writes or a result channel.
6. **Ignoring `ctx` in `fn`:** cancellation only stops *new* tasks; running ones finish.
7. **Unbuffered results channel with the consumer starting after `Wait()`:** deadlock. Either buffer `len(items)` or consume concurrently.
8. **Releasing the slot only on the success path:** a panic or early `return` leaks a permit and the pool slowly stalls. Release with `defer`.
9. **`time.After` in a hot loop:** each call allocates a timer. Prefer `time.NewTimer` plus `Stop` in long-running loops.
10. **Returning the derived ctx's error** after fail-fast, masking the real error. Return `firstErr`, and check `parent.Err()` separately.

---

## 11. Interview Question Bank (with answers)

**Q1. Concurrency vs parallelism? What does "bounded" protect?**
Concurrency is structuring work as independently progressing tasks; parallelism is literally running at the same time on multiple cores. Bounding protects scarce resources (memory, FDs, DB connections, downstream capacity), not CPU.

**Q2. How do you choose N?**
From the bottleneck: CPU-bound uses the core count; I/O-bound uses the downstream's limit, then measure and tune. Make it configurable with a sane default.

**Q3. Semaphore vs worker pool vs errgroup?**
Semaphore: simplest, one goroutine per task. Pool: long-lived workers, useful when they hold state or per-task goroutine overhead matters. errgroup: least code, but no panic recovery and `g.Go` needs a cancel check. Default to a semaphore or errgroup; build a pool only when workers need identity or state.

**Q4. Is the buffered channel's capacity the concurrency limit?**
Only in the semaphore pattern. In a pool, channel capacity is the *queue* size (how far the producer may run ahead); the concurrency limit is the worker count. Mixing them up is a common mistake.

**Q5. Design the API.**
`Map[T,R](ctx, items, N, fn) ([]R, error)` for simple use, with options (`WithRetry`, `WithTimeout`, `FailFast(false)`) via functional options (see [08-Functional-Options](../DESIGN-PATTERNS/08-Functional-Options.md)). For huge inputs, a channel-based streaming API. Generic over `T` and `R`; `ctx` first; `fn` takes `ctx`.

**Q6. Preserve input order?**
Carry the index and write to `results[idx]`. No sorting or locking needed; each element has a single writer.

**Q7. One task fails: cancel, wait, or partial results?**
Depends on semantics. All-or-nothing: cancel via derived ctx and return the first error. Batch: collect with `errors.Join` and return per-item results. Make it an option and document the default.

**Q8. Add retries and timeouts without breaking the bound?**
Decorate `fn` so the retry and the backoff sleep happen inside the worker slot. Use per-attempt `context.WithTimeout`. Retry only transient errors, with jitter and a cap.

**Q9. Add a rate limit on top?**
Separate limiter (`x/time/rate`) called inside the wrapper before `fn`. Concurrency and rate are independent constraints and both can be needed.

**Q10. Who closes which channel?**
The sole sender closes the jobs channel after the last send. The results channel (if used) is closed by a goroutine that waits on the `WaitGroup`. Receivers never close.

**Q11. How can this leak goroutines?**
A producer blocked on a send after consumers quit; workers blocked on a receive from a never-closed channel; tasks ignoring ctx. Fix: select on `ctx.Done()` for sends, close the jobs channel, pass ctx into `fn`.

**Q12. What if a worker panics?**
Without `recover`, the whole process crashes (a panic in any goroutine is fatal). Recover in the worker, convert to an error, release the slot via `defer`. Do not hide programmer errors in production; log with the stack.

**Q13. How do you prove the limit holds?**
Structural argument (acquire before spawn) plus an atomic peak-in-flight test under `-race`.

**Q14. Is `results[i] = v` from many goroutines a race?**
No. Different slice elements are different memory locations. Reading after `wg.Wait()` is safe due to happens-before. (Adjacent elements may share a cache line, which is a performance concern called false sharing, not a correctness one.)

**Q15. Do you still need `i := i` in Go 1.22+?**
No. The loop variable is per-iteration when `go.mod` declares Go 1.22 or later. Older modules still need the copy.

**Q16. Downstream starts failing; what do you do?**
Fail-fast and retry with backoff are not enough: add a circuit breaker to stop hammering, and adaptive concurrency (reduce N on errors or latency rise, AIMD style).

**Q17. A million items: avoid holding them all?**
Streaming API: producer reads from a source (file, DB cursor) into a bounded channel; workers pull; results written to a sink or output channel. Memory stays O(N + queue), not O(items).

**Q18. Shutdown on SIGTERM?**
`signal.NotifyContext` gives a ctx; stop intake; let in-flight tasks finish up to a deadline; then cancel; wait for the `WaitGroup`; report what was not processed so it can be resumed.

**Q19. Why not a goroutine per item plus a mutex-guarded counter?**
It creates all goroutines up front (memory cost) and reimplements a semaphore badly. Channels or `semaphore.Weighted` express the limit directly.

**Q20. How is this different from Pipeline?**
Bounded parallel processor: many workers doing the *same* step on independent items. Pipeline: different steps in sequence, each possibly with its own concurrency. They compose: each pipeline stage can be a bounded processor.

---

## 12. Practice Plan

1. Write `MapSem` from memory, with the peak-in-flight test.
2. Write `MapPool` from memory. Draw who sends, who receives, who closes.
3. Write `MapGroup`; reproduce the two gotchas in section 5 on purpose, then fix them.
4. Add `Retry` and `Paced`; test that retries do not exceed N in flight.
5. Make it streaming (`<-chan T` in, `<-chan Result` out) with index-tagged results.
6. Explain Q1 to Q20 aloud without looking.

## 13. Interview Check

Be ready to say: where N comes from, where the bound is enforced (acquire before spawn), how order is kept, the error policy and why, who closes what, how cancellation reaches both producer and running tasks, what happens on panic, and how you tested the bound under `-race`.
