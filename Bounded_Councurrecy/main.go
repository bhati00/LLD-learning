package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func main() {
	items := []int{1, 2, 3, 4, 5, 6}

	// Happy path: at most 2 tasks should overlap.
	res, err := SemaphorePattern(context.Background(), 2, items,
		func(ctx context.Context, x int) (int, error) {
			fmt.Println("start", x)
			time.Sleep(100 * time.Millisecond)
			fmt.Println("end  ", x)
			return x * 2, nil
		})
	fmt.Println(res, err)

	// Failure: item 3 fails, so the rest should stop.
	res, err = SemaphorePattern(context.Background(), 2, items,
		func(ctx context.Context, x int) (int, error) {
			if x == 3 {
				return 0, errors.New("boom")
			}
			select {
			case <-time.After(100 * time.Millisecond):
				return x, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		})
	fmt.Println(res, err)
}
