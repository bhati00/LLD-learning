package main

import (
	"context"
	"fmt"
	"slices"
)

func check(name string, ok bool) {
	if ok {
		fmt.Println("PASS:", name)
	} else {
		fmt.Println("FAIL:", name)
	}
}

func TestWorkerPool() {
	inputs := []int{1, 2, 3, 4, 5}
	square := func(ctx context.Context, n int) (int, error) { return n * n, nil }

	result, err := WorkerPool(context.Background(), inputs, square, 3)
	check("worker pool results in order", err == nil && slices.Equal(result, []int{1, 4, 9, 16, 25}))

	failOnFour := func(ctx context.Context, n int) (int, error) {
		if n == 4 {
			return 0, fmt.Errorf("i got error")
		}
		return n * n, nil
	}
	result, err = WorkerPool(context.Background(), inputs, failOnFour, 3)
	check("worker pool returns first error", err != nil && result == nil)

	_, err = WorkerPool(context.Background(), inputs, square, 0)
	check("worker pool rejects zero workers", err != nil)
}

func TestLru() {
	c := NewLruCache(2)
	c.add(1, "a")
	c.add(2, "b")
	c.get(1)
	c.add(3, "c")

	_, err := c.get(2)
	check("lru evicts least recently used", err != nil)

	v, err := c.get(1)
	check("lru keeps recently used", err == nil && v == "a")
}

func main() {
	TestWorkerPool()
	TestLru()
}
