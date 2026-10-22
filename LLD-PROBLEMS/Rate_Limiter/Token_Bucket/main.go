package main

import (
	"fmt"
	"sync"
)

func TestTockenBucket() {
	tb := NewTokenBucket(3, 1)
	var wg sync.WaitGroup

	for requestID := 1; requestID <= 4; requestID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			allowed := tb.Allow()
			fmt.Printf("Request %d: allowed=%t\n", id, allowed)
		}(requestID)
	}

	wg.Wait()
}
func main() {
	TestTockenBucket()
}
