package faninmultiplexer

import "context"

// Split fans out one input channel to n output channels.
// Each value goes to exactly one output (competing consumers), not to all.
// Every output closes once source is drained or ctx is cancelled.
func Split[T any](ctx context.Context, source <-chan T, n int) []<-chan T {
	outputs := make([]<-chan T, 0, n)

	for i := 0; i < n; i++ {
		ch := make(chan T)
		outputs = append(outputs, ch)

		go func(out chan<- T) {
			defer close(out)
			for v := range source {
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}

	return outputs
}
