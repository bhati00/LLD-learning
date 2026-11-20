package main

import (
	"fmt"
	"time"
)

func main() {
	timeout := 100 * time.Millisecond
	cb := NewCircuitBreaker(2, timeout, 2)

	fmt.Println("closed, allow:", cb.Allow(), "(want true)")

	cb.Record(true)
	cb.Record(true)
	fmt.Println("after 2 failures:", cb.State(), "(want 1 = open)")
	fmt.Println("allow while open:", cb.Allow(), "(want false)")

	time.Sleep(2 * timeout)
	fmt.Println("probe 1:", cb.Allow(), "(want true)")
	fmt.Println("state:", cb.State(), "(want 2 = half-open)")
	fmt.Println("probe 2:", cb.Allow(), "(want true)")
	fmt.Println("probe 3:", cb.Allow(), "(want false)")

	cb.Record(false)
	cb.Record(false)
	fmt.Println("after 2 probe successes:", cb.State(), "(want 0 = closed)")
}
