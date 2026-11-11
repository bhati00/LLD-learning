package main

import (
	"errors"
	"sync"
)

/*
Broker : maintains the topic map with their subscribers.
*/
type Message struct {
	payload string
}

type Subscription struct {
	listener chan Message
	id       uint64
	topic    string
}

type Broker struct {
	topic map[string]map[uint64]*Subscription
	mu    sync.Mutex
	count uint64 // used for incremention of IDs
}

func NewBroker() *Broker {
	return &Broker{
		topic: make(map[string]map[uint64]*Subscription),
	}
}

/*
it fetches the all subscription for some topic
than push the messages to their channels
*/
func (b *Broker) Publish(message string, topic string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	val, ok := b.topic[topic]
	if !ok {
		return
	}
	for _, sub := range val {
		select {
		case sub.listener <- Message{
			payload: message,
		}:
		default:
		}
	}
}

/*
create the key-value pair in broker
use the counter to increment the IDs
susscription should have buffered channel for listning the data
*/
func (b *Broker) Subscribe(topic string) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	innermap, ok := b.topic[topic]
	if !ok {
		innermap = make(map[uint64]*Subscription)
		b.topic[topic] = innermap
	}
	b.count += 1
	sub := Subscription{
		listener: make(chan Message, 10),
		id:       b.count,
		topic:    topic,
	}
	innermap[b.count] = &sub
	return &sub
}

func (b *Broker) Unsubscribe(topic string, subId uint64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	exist, ok := b.topic[topic]
	if !ok {
		return errors.New("fsfsd")
	}
	culprit, ok := exist[subId]
	if !ok {
		return errors.ErrUnsupported
	}
	close(culprit.listener)
	delete(exist, subId)
	if len(exist) == 0 {
		delete(b.topic, topic)
	}
	return nil
}
