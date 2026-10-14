package main

import (
	"context"
	"fmt"
)

func main() {
	inputs := []int{1, 2, 3, 4, 5}
	result, err := WorkerPool(context.Background(), inputs, func(ctx context.Context, num int) (int, error) {

		return num * num, nil
	}, 3)
	fmt.Println(result, err)
}
