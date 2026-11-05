package main

import (
	"context"
	"fmt"
	"sync"
)

/*
input data -> source channel (generator)
-> fan-out : N worker goroutines read form the channel
-> each worker pushes its result to an output channel
-> fan-in : merge the worker outputs into one channel
-> consumer reads the merged channel and collects result in one slice
**/

func generator(ctx context.Context, input []int) <-chan int {
	in := make(chan int)
	go func() {
		defer close(in)
		for _, val := range input {
			select {
			case <-ctx.Done():
				return
			case in <- val:
			}
		}
	}()
	return in
}

func worker(ctx context.Context, in <-chan int) <-chan int {
	output := make(chan int)
	go func() {
		defer close(output)
		for val := range in {
			result := val * val
			select {
			case <-ctx.Done():
				return
			case output <- result:
			}
		}
	}()
	return output
}

func fanIn(ctx context.Context, inputs ...<-chan int) <-chan int {
	output := make(chan int)
	var wg sync.WaitGroup
	wg.Add(len(inputs))
	for _, ch := range inputs {
		go func() {
			defer wg.Done()
			for val := range ch {
				select {
				case <-ctx.Done():
					return
				case output <- val:
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(output)
	}()
	return output
}
func fanOut(ctx context.Context, input <-chan int, n int) []<-chan int {
	workers := make([]<-chan int, 0, n)
	for range n {
		workers = append(workers, worker(ctx, input))
	}
	return workers
}
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	data := []int{1, 2, 3, 4, 4}
	src := generator(ctx, data)
	merged := fanIn(ctx, fanOut(ctx, src, 3)...)
	var result []int
	for v := range merged {
		result = append(result, v)
	}
	fmt.Println(result)

}
