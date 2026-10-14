package main

import (
	"context"
	"sync"
)

// Merge fans in all input channels into one output channel.
// The output closes after every input is drained or ctx is cancelled.
func Merge[T any](ctx context.Context, chans ...<-chan T) <-chan T {
	output := make(chan T)

	var wg sync.WaitGroup
	wg.Add(len(chans))

	for _, ch := range chans {
		go func(in <-chan T) {
			defer wg.Done()
			for v := range in {
				select {
				case output <- v:
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}

	// Only one closer: runs after every worker has finished.
	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}
