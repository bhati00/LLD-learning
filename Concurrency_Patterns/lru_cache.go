package main

import (
	"fmt"
	"sync"
)

// create a map of capacity c if it increased remove the least recent used one
// IDEA : maintain a map for the search loop up in O(1), use the doubly linked list for deletion in O(1)
// cache struct will maintain the 2 pointers, tail and head representing mru and the lru cache
// if a cache is access move it forward means adding to the tail resulting in the head as the lru cache

type Node struct {
	next *Node
	prev *Node
	val  string
	key  int
}

func NewNode(val string) *Node {
	return &Node{
		next: nil,
		prev: nil,
		val:  val,
	}
}

type LruCache struct {
	head     *Node
	tail     *Node
	cache    map[int]*Node
	mu       sync.Mutex
	capacity int
}

func NewLruCache(capacity int) *LruCache {
	lru := &LruCache{
		head:     NewNode(""),
		tail:     NewNode(""),
		cache:    make(map[int]*Node),
		capacity: capacity,
	}
	lru.head.next = lru.tail
	lru.tail.prev = lru.head
	return lru
}

func unlink(node *Node) {
	node.next.prev = node.prev
	node.prev.next = node.next
}

// adding the node the tail
func (lru *LruCache) AddForward(node *Node) {
	node.prev = lru.tail.prev
	node.next = lru.tail
	lru.tail.prev.next = node
	lru.tail.prev = node
}

func (lru *LruCache) PushForward(node *Node) {
	unlink(node)
	lru.AddForward(node)
}

// get the cache val = move node to forward
func (lru *LruCache) get(key int) (string, error) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	node, exists := lru.cache[key]
	if exists {
		lru.PushForward(node)
		return node.val, nil
	} else {
		return "", fmt.Errorf("value not found")
	}
}

// update the cache val = add node to the forward , if capacity crossed delete the head
func (lru *LruCache) add(key int, val string) bool {
	if lru.capacity <= 0 {
		return false
	}
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if node, exists := lru.cache[key]; exists {
		node.val = val
		lru.PushForward(node)
		return true
	}
	node := NewNode(val)
	node.key = key
	lru.cache[key] = node
	lru.AddForward(node)
	if len(lru.cache) > lru.capacity {
		lruNode := lru.head.next
		unlink(lruNode)
		delete(lru.cache, lruNode.key) // remove from the map
	}
	return true
}
