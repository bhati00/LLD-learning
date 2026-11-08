package main

import (
	"fmt"
)

func check(name string, ok bool) {
	if ok {
		fmt.Println("PASS:", name)
	} else {
		fmt.Println("FAIL:", name)
	}
}

func TestLru() {
	c := NewLruCache(2)
	c.add(1, 1)
	c.add(2, 1)
	c.get(1)
	c.add(3, 1)

	_, err := c.get(2)
	check("lru evicts least recently used", err != nil)

	v, err := c.get(1)
	check("lru keeps recently used", err == nil && v == 1)
}

func main() {
	TestLru()
}
