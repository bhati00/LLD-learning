package main

import (
	"errors"
	"sync"
)

type Message struct {
	payload string
}

type PublishResult struct {
	Delivered int
	Dropped   int
	Skipped   int
}

type Subscription struct {
	topic   string
	id      uint64 // can be used to remove a subscriber quickly
	message chan Message
	mu      sync.Mutex // used for safely unsubscribing without sending on the close
	closed  bool
}

type Broker struct {
	topics map[string]map[uint64]*Subscription
	mu     sync.Mutex
	nextId uint64 // helps to assign the ID using the counter
}

func NewBroker() *Broker {
	return &Broker{
		topics: make(map[string]map[uint64]*Subscription),
	}
}

// Subscribers : downstream subscribes to the broker and listens via message buffered channel
func (broker *Broker) Subscribe(topic string) (*Subscription, error) {
	if topic == "" {
		return nil, errors.New("topic must not be empty")
	}

	broker.mu.Lock()
	defer broker.mu.Unlock()

	if _, ok := broker.topics[topic]; !ok {
		broker.topics[topic] = make(map[uint64]*Subscription)
	}

	broker.nextId++
	subscription := &Subscription{
		topic:   topic,
		id:      broker.nextId,
		message: make(chan Message, 10),
	}
	broker.topics[topic][subscription.id] = subscription

	return subscription, nil
}

// Publish : upstream pushes data to the topics with message, than we pass that data into subscribers listen channel
/* NOTE :“I hold the mutex only while accessing the subscription registry.
 Channel sends can block, so holding the registry lock during delivery would let one slow subscriber block unrelated broker operations
**/
func (broker *Broker) Publish(topic string, message string) PublishResult {
	var result PublishResult
	broker.mu.Lock()

	subs, ok := broker.topics[topic]
	if !ok || len(subs) == 0 {
		broker.mu.Unlock()
		return result
	}

	// copy the pointers before unlocking
	snapshot := make([]*Subscription, 0, len(subs))
	for _, sub := range subs {
		snapshot = append(snapshot, sub)
	}

	broker.mu.Unlock()

	// deliver the message
	for _, sub := range snapshot {
		sub.mu.Lock()
		if !sub.closed {
			select {
			case sub.message <- Message{payload: message}:
				result.Delivered++
			default:
				result.Dropped++
			}
		} else {
			result.Skipped++
		}
		sub.mu.Unlock()
	}
	return result
}

func (broker *Broker) Unsubscribe(subId uint64, topicName string) (bool, error) {
	broker.mu.Lock()
	topic := broker.topics[topicName]
	sub, ok := topic[subId]
	if !ok {
		broker.mu.Unlock()
		return false, errors.New("subscription not found")
	}
	delete(topic, subId)
	broker.mu.Unlock()
	sub.mu.Lock()
	sub.closed = true
	close(sub.message)
	sub.mu.Unlock()
	return true, nil
}
