# Pipeline Pattern

> **Process data through a sequence of stages, where each stage transforms or forwards values.**

Pipelines make multi-step processing easier to compose. In Go, stages commonly communicate over channels, but a simple sequence of functions may be better for synchronous work.

---

## The Core Idea

Each stage has a clear input and output. A stage can transform, filter, or enrich data; later stages consume its output. A concurrent pipeline connects stages with channels and goroutines.

---

## Example: Concurrent Number Pipeline

```go
package pipeline

func generate(values ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, value := range values {
			out <- value
		}
	}()
	return out
}

func square(input <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for value := range input {
			out <- value * value
		}
	}()
	return out
}

func main() {
	for result := range square(generate(1, 2, 3, 4)) {
		println(result)
	}
}
```

The receiver drains the final channel; ranging it also observes closure. In real systems, cancellation must be part of the design so a downstream stage that stops early does not leave upstream goroutines blocked forever.

---

## Cancellation and Errors

For production pipelines, stages should usually accept `context.Context` and select between sending and cancellation:

```go
select {
case out <- result:
case <-ctx.Done():
	return
}
```

Choose an explicit error strategy: a result struct containing value and error, a separate error channel, or a stage function that returns an error. Ensure every goroutine has a clear owner and shutdown path.

---

## When It Helps

- Work naturally consists of independent processing stages.
- Stages can be reused or rearranged.
- Bounded concurrency or streaming avoids loading all input into memory.

## Costs and Pitfalls

- Every channel and goroutine adds coordination and shutdown complexity.
- Unbuffered stages may limit throughput; unbounded buffering risks memory exhaustion.
- Early return can leak goroutines unless cancellation is propagated.
- Multiple producers require a clear rule for who closes a channel.
- Not every sequential transformation benefits from concurrency; function composition may be simpler.

## Interview Check

Describe ownership and closure for each channel, how cancellation propagates, how errors travel, and how backpressure limits memory. Compare a pipeline with a worker pool: a pipeline has distinct processing stages; a worker pool applies a bounded set of workers to similar jobs.