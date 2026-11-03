package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Pipeline: generate -> square (fan-out to N workers) -> merge (fan-in) -> print
// Rules: every stage owns its output channel, closes it when done, and stops on ctx.Done().

// Stage 1: source. Emits values, then closes.
func generate(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Stage 2: processing. Reads from in, writes processed values to out.
func square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			time.Sleep(100 * time.Millisecond) // simulate slow work
			select {
			case out <- n * n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Fan-out: start `workers` copies of the stage, all reading the SAME input channel.
// Each value is picked up by exactly one worker.
func fanOut(ctx context.Context, in <-chan int, workers int) []<-chan int {
	outs := make([]<-chan int, workers)
	for i := 0; i < workers; i++ {
		outs[i] = square(ctx, in)
	}
	return outs
}

// Fan-in: merge many channels into one. Close the merged channel only after
// ALL inputs are drained (WaitGroup).
func fanIn(ctx context.Context, chans ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	wg.Add(len(chans))

	for _, c := range chans {
		go func(c <-chan int) {
			defer wg.Done()
			for v := range c {
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // stops all goroutines if we exit early

	start := time.Now()

	src := generate(ctx, 1, 2, 3, 4, 5, 6, 7, 8)
	workers := fanOut(ctx, src, 4)
	merged := fanIn(ctx, workers...)

	sum := 0
	for v := range merged {
		fmt.Println("result:", v) // order is NOT guaranteed
		sum += v
	}

	fmt.Println("sum:", sum, "took:", time.Since(start).Round(100*time.Millisecond))
}
