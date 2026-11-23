package main

import (
	"context"
	"errors"
	"sync"
)

var ErrQueueFull = errors.New("notification queue is full")

// delivery is one unit of work: one message to one recipient on one channel.
// results belongs to the Send call that created it.
type delivery struct {
	ctx       context.Context
	recipient Recipient
	channel   Channel
	chanType  ChannelType
	subject   string
	body      string
	results   chan<- Result
}

type NotificationService struct {
	channels    map[ChannelType]Channel // registered channels
	users       map[int]Recipient       // user registry
	workerCount int
	jobs        chan delivery
	wg          sync.WaitGroup
	stopOnce    sync.Once
}

func NewNotificationService(workerCount, queueSize int) *NotificationService {
	return &NotificationService{
		channels:    make(map[ChannelType]Channel),
		users:       make(map[int]Recipient),
		workerCount: workerCount,
		jobs:        make(chan delivery, queueSize),
	}
}

// RegisterChannel and AddUser are not thread-safe: call them before Start.
func (ns *NotificationService) RegisterChannel(t ChannelType, ch Channel) {
	ns.channels[t] = ch
}

func (ns *NotificationService) AddUser(r Recipient) {
	ns.users[r.id] = r
}

// Start launches the worker pool. Call it once.
func (ns *NotificationService) Start() {
	for i := 0; i < ns.workerCount; i++ {
		ns.wg.Add(1)
		go ns.worker()
	}
}

func (ns *NotificationService) worker() {
	defer ns.wg.Done()
	for d := range ns.jobs {
		err := d.ctx.Err()
		if err == nil {
			err = d.channel.send(d.ctx, d.recipient, d.subject, d.body)
		}
		// results is buffered for every delivery of the call, so this never blocks.
		d.results <- Result{userID: d.recipient.id, channel: d.chanType, err: err}
	}
}

// Stop closes the queue and waits for the workers to drain it.
// Call it after the last Send: a Send after Stop would panic.
func (ns *NotificationService) Stop() {
	ns.stopOnce.Do(func() { close(ns.jobs) })
	ns.wg.Wait()
}

// Send queues one delivery per (user, channel) pair and returns immediately.
// The returned channel is buffered and receives exactly one Result per pair
// (ErrQueueFull if the pair couldn't be queued). It is not closed, so read
// len(n.userIDs)*len(n.channels) results, or ignore it. Users and channels
// are not validated: they must be registered.
func (ns *NotificationService) Send(ctx context.Context, n Notification) <-chan Result {
	results := make(chan Result, len(n.userIDs)*len(n.channels))

	for _, id := range n.userIDs {
		for _, t := range n.channels {
			d := delivery{
				ctx:       ctx,
				recipient: ns.users[id],
				channel:   ns.channels[t],
				chanType:  t,
				subject:   n.subject,
				body:      n.body,
				results:   results,
			}
			select {
			case ns.jobs <- d:
			default:
				results <- Result{userID: id, channel: t, err: ErrQueueFull}
			}
		}
	}
	return results
}
