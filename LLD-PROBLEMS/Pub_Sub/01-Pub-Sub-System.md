# Pub/Sub System: Go Interview Roadmap

**For:** Go developers preparing for 3-4 YOE interviews.

Follow this order. Each topic's theory and implementation will be covered separately.

## What interviewers ask

The base problem is small: publishers publish to topics, subscribers receive every message on topics they subscribed to, many of each, thread-safe. Typical classes: `Message`, `Topic`, `Subscriber` (interface), `Publisher`, `Broker`.

At 3-4 YOE that is the warm-up. The follow-ups decide the round:

- A subscriber is slow. Does it block the publisher or the other subscribers?
- A subscriber unsubscribes while a publish is running. Is that safe?
- How does `Close()` stop everything without a panic or a goroutine leak?
- Add wildcard topics, consumer groups, ack and retry, or replay.

## Roadmap

| Order | Topic | Interview weight | Material |
|---|---|---|---|
| 1 | **Theory:** pub/sub vs. Observer vs. queue; push vs. pull; delivery semantics (at-most-once, at-least-once, exactly-once) and ordering; how Redis, NATS, Kafka and Google Pub/Sub differ; clarifying questions to ask | Opening 5-10 minutes, plus the scale-up answer | [Theory and diagrams](Foundations/01-Foundations.md) |
| 2 | **Core broker (must code):** `Subscribe`, `Unsubscribe`, `Publish`; topic registry; bounded per-subscription delivery; slow-subscriber policy; `Close()`; tests with `-race` | The problem itself | [Interview problem statement](Core_Broker/01-Core-Broker.md); implementation together |
| 3 | **Routing extensions:** wildcard topics (`*` and `>` via a token trie) and queue/consumer groups (one-of-N delivery, key-based ordering) | Likely follow-up | Planned |
| 4 | **Reliability and replay:** ack/nack, ack deadline, retry with backoff, dead-letter topic, idempotent subscribers; append-only log with per-subscription offsets for late joiners | Likely follow-up; sketch if time allows | Planned |

For each implementation, cover configuration validation, Go concurrency safety, and focused tests before moving on.

## Already covered elsewhere (not repeated here)

| Needed for | Where |
|---|---|
| Channels, `select`, closed-channel behavior, goroutine leaks | Go Learning, Phase 9 (Concurrency Depth) |
| `RWMutex`, `WaitGroup`, `context` cancellation | Go Learning, Phase 9 |
| Race detector | Go Learning, Phase 10 |
| Observer, Worker Pool, Fan-in | [DESIGN-PATTERNS/](../../DESIGN-PATTERNS), [Concurrency_Patterns/](../../Concurrency_Patterns) |

## Interview question bank

- Pub/sub vs. Observer: what changes when a broker sits between publisher and subscriber?
- Pub/sub vs. a message queue: does every subscriber get each message, or only one consumer?
- Why not hold the registry lock while delivering to subscribers?
- One goroutine per subscriber or a shared worker pool? What does a full subscriber buffer do?
- What ordering do you guarantee: per topic, per subscriber, or none?
- A subscriber crashes mid-message. What happens under at-most-once and at-least-once?
- A subscriber joins late. Does it see earlier messages?

## Sources

- Reference LLD spec: [ashishps1/awesome-low-level-design, Pub-Sub problem](https://github.com/ashishps1/awesome-low-level-design/blob/main/problems/pub-sub-system.md)
- Delivery semantics: [Redis Pub/Sub](https://redis.io/docs/latest/develop/pubsub/), [NATS core](https://docs.nats.io/learn/core-nats/), [NATS queue groups](https://docs.nats.io/learn/core-nats/queue-groups), [Google Pub/Sub subscriptions](https://docs.cloud.google.com/pubsub/docs/subscription-overview)
- Partitions and consumer groups: [Hello Interview, Kafka deep dive](https://www.hellointerview.com/learn/system-design/deep-dives/kafka)

The follow-up list is a synthesis of the spec and of how these systems behave, not a transcript of any company's interview.
