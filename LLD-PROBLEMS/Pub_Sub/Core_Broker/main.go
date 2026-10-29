package main

import (
	"fmt"
	"log"
	"sync"
)

func main() {
	broker := NewBroker()
	sub, err := broker.Subscribe("orders")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("First publish:", broker.Publish("orders", "order created"))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for msg := range sub.message {
			fmt.Println("Received:", msg.payload)
		}
	}()
	go func() {
		defer wg.Done()
		fmt.Println("Concurrent publish:", broker.Publish("orders", "order deleted"))
	}()

	if _, err := broker.Unsubscribe(sub.id, sub.topic); err != nil {
		log.Fatal(err)
	}
	wg.Wait()
	fmt.Println("Done.")
}
