# Observer Pattern

> **Notify interested subscribers when a subject publishes an event or changes.**

Observer decouples the publisher from the code that reacts. In Go services, this often appears as callbacks, event handlers, or a pub/sub abstraction.

---

## The Core Idea

The subject emits an event without depending on specific consumers. Subscribers decide what to do with it. This is useful when one business event has several independent reactions, such as recording an audit event and sending a notification.

---

## Minimal Synchronous Example

```go
package events

type OrderPlaced struct {
	OrderID string
}

type Handler func(OrderPlaced)

type Publisher struct {
	handlers []Handler
}

func (p *Publisher) Subscribe(handler Handler) {
	p.handlers = append(p.handlers, handler)
}

func (p *Publisher) Publish(event OrderPlaced) {
	for _, handler := range p.handlers {
		handler(event)
	}
}
```

The publisher knows only that handlers accept `OrderPlaced`. This simple version is intentionally synchronous: handler latency and panics affect the publisher.

---

## Production Decisions

Before making delivery concurrent or durable, specify the contract:

- Is delivery synchronous or asynchronous?
- Does a slow handler block the publisher?
- Are events delivered at-most-once, at-least-once, or best-effort?
- What happens when a handler fails: retry, skip, or return an error?
- Can subscriptions be removed, and what happens during concurrent publish/unsubscribe?
- Is event ordering guaranteed?

An in-process observer is not a durable message broker. If events must survive process failure or cross service boundaries, use a persistent queue or an outbox-based design.

---

## When It Helps

- Multiple independent components react to the same event.
- The publisher should not import or call concrete consumers.
- Adding a new reaction should not change the business operation that emitted the event.

## Costs and Pitfalls

- Hidden control flow makes it harder to know what a publish operation will trigger.
- Synchronous handlers couple availability and latency.
- Asynchronous delivery requires explicit concurrency, error, ordering, and shutdown policies.
- Unbounded goroutine-per-event designs can exhaust resources.
- Avoid broad generic event systems when a direct function call is simpler and clearer.

## Interview Check

State the delivery guarantees and failure behavior, not just the subscriber interface. Compare in-process Observer with broker-backed pub/sub in terms of durability, coupling, and operational complexity.