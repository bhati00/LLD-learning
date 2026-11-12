# Core Pub/Sub Broker: Interview Problem

[Back to the roadmap](../01-Pub-Sub-System.md) | [Foundations](../Foundations/01-Foundations.md)

**Target:** 3-4 YOE Go interviews.

This is a practice interview prompt adapted from the [reference Pub/Sub problem](https://github.com/ashishps1/awesome-low-level-design/blob/main/problems/pub-sub-system.md), with explicit concurrency and lifecycle requirements. It is not a verbatim company question.

No implementation or solution is included. We will clarify the contract, design the API, and implement it together.

## 1. The Problem Statement

> Design and implement an in-memory, topic-based publish/subscribe broker in Go.
>
> Multiple publishers can publish messages to named topics. Subscribers can subscribe to topics and receive messages published after they subscribe. A topic can have multiple independent subscriptions.
>
> Support subscribing, publishing, unsubscribing, and closing the broker. Operations must be safe when called concurrently. Handle slow subscribers without allowing memory or background work to grow without bound.
>
> Demonstrate the behavior with focused tests.

Example use case: an order service publishes an `OrderCreated` event. Both the email service and analytics service need their own delivery of that event. They must not compete for it.

## 2. Scope and Required Behavior

| Area | Requirement |
|---|---|
| Execution | One Go process; all state is in memory |
| Topics | Exact, case-sensitive names; no wildcard matching |
| Subscription | Each successful subscribe creates an independent subscription to one topic, even if another subscription has the same consumer |
| Publishing | Attempt delivery to every active subscription on the named topic |
| Isolation | A subscription receives only messages for its topic |
| History | No replay: a new subscription does not receive messages published before it joined |
| Capacity | Each subscription has a finite pending-message capacity; define what happens when it fills |
| Delivery | At-most-once delivery attempts per publish call per subscription; no automatic retries |
| Ordering | For sequential publishes to a topic, each subscription observes accepted messages in that order; dropped messages may create gaps |
| Unsubscribe | Stops future delivery to that subscription without affecting other subscriptions |
| Shutdown | Closing stops admission of new work and releases broker-owned resources |
| Concurrency | Publish, subscribe, unsubscribe, and close may overlap without data races, deadlocks, or lifecycle panics |

Two publish calls with identical payloads are still two separate publications. Deduplication is not required.

Here, **delivery** means handing a message to the subscription's delivery mechanism, not successfully completing the consumer's business operation. A successful publish must not claim that all handlers finished.

## 3. Clarify Before Coding

These are intentional discussion points, not extra features. We will settle one contract for each before implementation rather than implement every option.

| Decision | What we need to agree on |
|---|---|
| Receiving messages | Does the subscriber read a stream, or does the broker invoke a registered handler? |
| Topic creation | Are topics created automatically on subscribe, or through an explicit operation? What happens on publish to an unknown topic? |
| Input validation | Which topic names and message values are valid? How is invalid capacity reported? |
| Slow subscriber | Block with cancellation/deadline, drop the newest message, or another bounded policy? Must healthy subscribers continue independently? |
| Publish outcome | How does the caller distinguish accepted deliveries, dropped deliveries, no subscribers, and broker errors? |
| Concurrent ordering | Are overlapping publishes unordered relative to one another, or must every subscription observe a common topic order? |
| Unsubscribe boundary | Can an already-started delivery finish after unsubscribe returns? Can already-buffered messages still be consumed? |
| Shutdown policy | Drain accepted messages or discard pending ones? How can shutdown finish if a consumer stops reading? |
| Repeated operations | What happens on repeated unsubscribe and repeated close? What do operations after close return? |

Do not silently drop messages or leave failures unobservable. The chosen policy must have an explicit outcome visible to the caller or subscriber.

## 4. Example Scenarios

### A. Broadcast and topic isolation

1. Email and Analytics subscribe independently to `orders.created`.
2. Audit subscribes to `payments.completed`.
3. Publish message M1 to `orders.created`.
4. With capacity available, Email and Analytics each receive M1 once. Audit receives nothing.

### B. No history and unsubscribe

1. Email subscribes to `orders.created`; publish M1 and allow its delivery to finish.
2. Analytics now subscribes to `orders.created`.
3. Publish M2 and allow its delivery to finish.
4. Email has M1 and M2; Analytics has only M2.
5. Unsubscribe Email, then publish M3.
6. Analytics receives M3; Email does not.

### C. Slow subscriber

1. Email and Analytics subscribe to the same topic.
2. Email stops consuming while Analytics continues.
3. Publish enough messages to fill Email's pending capacity.
4. Verify the policy we chose: blocking must be cancellable, or overflow must be reported. Memory and background work must remain bounded.
5. Verify the promised behavior for Analytics; isolation from Email is a requirement only if we choose a non-blocking fan-out contract.

### D. Lifecycle races

1. Publish while another caller unsubscribes.
2. Close while callers are publishing or subscribing.
3. All calls finish according to the agreed contract, without a panic or data race.
4. New work after shutdown is rejected explicitly.

## 5. Acceptance Checklist

- [ ] Multiple topics, publishers, and independent subscriptions work.
- [ ] Broadcast is not accidentally implemented as competing consumers.
- [ ] Topic isolation and no-history behavior match scenarios A and B.
- [ ] Sequential message ordering is preserved for accepted deliveries.
- [ ] Capacity and slow-subscriber behavior match the chosen contract.
- [ ] Unsubscribe affects only the intended subscription.
- [ ] Concurrent operations respect the agreed unsubscribe and close boundaries.
- [ ] Invalid inputs, repeated operations, and post-close calls have defined outcomes.
- [ ] Shutdown finishes under the agreed policy, including when a consumer stops reading.
- [ ] Tests coordinate relevant events rather than relying on arbitrary sleeps, use bounded waits, and pass with Go's race detector.

## 6. Out of Scope

Do not add wildcard routing, consumer groups, acknowledgements, retries, dead-letter topics, persistence, replay, networking, or distributed brokers. Those belong to later roadmap topics.

We will apply the Go concurrency concepts already studied, not repeat their internals here.

**Our next step:** answer the clarification table and propose the public API and its guarantees. Only then start coding.

## Current Implementation and Demo

[broker.go](broker.go) now supports subscribing, publishing, and unsubscribing. Delivery is non-blocking: a full subscription buffer drops the newest delivery attempt. `PublishResult` reports queued deliveries (`Delivered`), full-buffer drops (`Dropped`), and snapshot recipients closed before delivery (`Skipped`). Queuing does not mean business processing has completed.

Unsubscribe removes the subscription under the broker lock, then closes its channel under the subscription lock. Publishers use that same subscription lock for checking closed state and attempting a send. Buffered messages remain readable after unsubscribe.

[main.go](main.go) is a small manual demo with one subscription. It queues an initial message, starts a consumer and another publish in goroutines, and unsubscribes from the main goroutine. One wait group waits for the reader and publisher to finish. The second message may or may not be queued, depending on scheduling. This prints delivery outcomes but does not assert correctness or test broadcast or overflow. Run from this directory:

```powershell
go run -race broker.go main.go
```

Counts vary with scheduling. No race report means none was detected in that execution, not a proof of race freedom. Concurrent publishers do not currently guarantee a common order across subscriptions; broker-wide `Close` is not implemented yet.
