package main

import (
	"errors"
	"sync"
)

// LRU cache : just hashmap with capacity, when it's exceeded remove the least used values
// Implementation Idea : just use the map for the quick lookup, but we need to remove the stale
// entry too in O(1) when capacity exceeded. for that we'll use DLL supports deletion
// in constant. so, when the data accessed dll will push that node to front. that way
// stale node is always in the head side.
// while removing the node : linked list node should have key to access the map's stale pair
// cache has to maintain tail and head representing the MRU and the LRU

var ErrNotFound = errors.New("key not found")

type linkedList struct {
	val  int
	next *linkedList
	prev *linkedList
	key  int
}

func newLinkedList(val, key int) *linkedList {
	return &linkedList{
		val: val,
		key: key,
	}
}

type LruCache struct {
	mu       sync.Mutex
	cacheMap map[int]*linkedList
	capacity int
	tail     *linkedList
	head     *linkedList
}

func NewLruCache(capacity int) *LruCache {
	lrucache := LruCache{
		capacity: capacity,
		tail:     newLinkedList(0, 0),
		head:     newLinkedList(0, 0),
		cacheMap: map[int]*linkedList{},
	}
	lrucache.tail.prev = lrucache.head
	lrucache.head.next = lrucache.tail
	return &lrucache
}

// delete the node
func deleteNode(node *linkedList) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// add to the front
func (lru *LruCache) addFront(node *linkedList) {
	node.prev = lru.tail.prev
	node.next = lru.tail
	lru.tail.prev.next = node
	lru.tail.prev = node
}

// push node forward = remove from the current pos + add to the front
func (lru *LruCache) pushForward(node *linkedList) {
	deleteNode(node)
	lru.addFront(node)
}

// access cache = get operation + push that node forward
func (lru *LruCache) get(key int) (int, error) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	node, ok := lru.cacheMap[key]
	if !ok {
		return 0, ErrNotFound
	}
	lru.pushForward(node)
	return node.val, nil
}

// add or update cache
func (lru *LruCache) add(key, val int) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	if lru.capacity <= 0 {
		return
	}
	if node, ok := lru.cacheMap[key]; ok {
		node.val = val
		lru.pushForward(node)
		return
	}
	newNode := newLinkedList(val, key)
	lru.cacheMap[key] = newNode
	lru.addFront(newNode)
	// if cap exceeded
	if len(lru.cacheMap) > lru.capacity {
		lruNode := lru.head.next
		delete(lru.cacheMap, lruNode.key)
		deleteNode(lruNode)
	}
}
