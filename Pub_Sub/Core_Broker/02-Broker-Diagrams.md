# Core Broker: Whole-System Diagrams

[Back to the problem statement](01-Core-Broker.md)

Visual map of what we are building. It uses your current structs in [types.go](types.go). Open in a Markdown preview to render the diagrams.

Open decisions (buffer-full policy, unsubscribe and close boundaries) are marked **TBD** and will be settled when we implement each operation.

## 1. The Big Picture

```mermaid
flowchart LR
    subgraph Publishers
        OS[Order service]
        PS[Payment service]
    end

    subgraph Broker["Broker (mu, topics, nextID)"]
        direction TB
        subgraph T1["topic: orders"]
            S1["Subscription 1<br/>id=1, topic=orders<br/>messages chan"]
            S2["Subscription 2<br/>id=2, topic=orders<br/>messages chan"]
        end
        subgraph T2["topic: payments"]
            S3["Subscription 3<br/>id=3, topic=payments<br/>messages chan"]
        end
    end

    subgraph Consumers
        EM[Email]
        AN[Analytics]
        AU[Audit]
    end

    OS -- "Publish(orders, msg)" --> Broker
    PS -- "Publish(payments, msg)" --> Broker

    S1 -- "reads from channel" --> EM
    S2 -- "reads from channel" --> AN
    S3 -- "reads from channel" --> AU
```

Reading it left to right:

- Publishers hand a message and a topic name to the broker.
- The broker looks up the subscriptions registered under that topic.
- Each subscription has its own channel, so every subscriber of the topic gets its own copy.
- Consumers read from their own channel at their own speed.

## 2. Structure: Class Diagram

```mermaid
classDiagram
    class Broker {
        mu : RWMutex
        topics : map~string, map~uint64, *Subscription~~
        nextID : uint64
        Subscribe(topic) Subscription
        Publish(topic, msg)
        Unsubscribe(sub)
        Close()
    }

    class Subscription {
        id : uint64
        topic : string
        messages : chan Message
    }

    class Message {
        orderEvent : eventType
        payload : string
    }

    class eventType {
        <<enumeration>>
        CREATED
        DELETED
    }

    Broker "1" o-- "0..*" Subscription : registers, grouped by topic name
    Subscription ..> Message : delivers through its channel
    Message --> eventType : has
```

The four methods on `Broker` are the planned API, not yet written.

How to read the `topics` field:

```text
topics = {
  "orders":   { 1: Subscription 1, 2: Subscription 2 },
  "payments": { 3: Subscription 3 }
}
        ^topic name     ^subscription id    ^the subscription
```

## 3. Operation: Subscribe

```mermaid
sequenceDiagram
    participant C as Consumer (Email)
    participant B as Broker
    participant R as topics registry

    C->>B: Subscribe("orders")
    B->>B: lock mu
    B->>B: nextID++ (get id=1)
    B->>B: create Subscription with its own buffered channel
    B->>R: topics["orders"][1] = subscription
    Note over R: create the inner map first if "orders" is new
    B->>B: unlock mu
    B-->>C: return Subscription 1
    C->>C: start reading Subscription.messages
```

Why a write lock: we change the map and `nextID`, so no other caller may read or write them at the same time.

## 4. Operation: Publish

```mermaid
sequenceDiagram
    participant P as Publisher (Order service)
    participant B as Broker
    participant S1 as Subscription 1 (Email)
    participant S2 as Subscription 2 (Analytics)

    P->>B: Publish("orders", {CREATED, "order-101"})
    B->>B: read-lock mu
    B->>B: copy subscriptions under "orders" into a local list
    B->>B: unlock mu
    Note over B: deliver after unlocking, so a slow<br/>subscriber cannot hold up the registry
    B->>S1: send message into messages channel
    B->>S2: send message into messages channel
    B-->>P: return outcome (TBD: delivered / dropped / no subscribers)
```

Both subscriptions get the message because each one has its own channel. One shared channel with two readers would split the messages instead of copying them.

## 5. Operation: What If a Subscriber Is Slow?

```mermaid
flowchart TD
    A["Broker sends message to a subscription's channel"] --> B{"Channel has free space?"}
    B -- Yes --> C["Message is queued. Consumer reads it later"]
    B -- No --> D{"Chosen policy (TBD)"}
    D -- "Block, with cancel or timeout" --> E["Publisher waits, then gives up with an error"]
    D -- "Drop newest" --> F["Message is skipped for this subscription and the caller is told"]
```

Whichever policy we choose, the buffer is finite and the outcome must be visible to the caller. That is what keeps memory bounded.

## 6. Operation: Unsubscribe

```mermaid
sequenceDiagram
    participant C as Consumer (Email)
    participant B as Broker

    C->>B: Unsubscribe(Subscription 1)
    B->>B: lock mu
    B->>B: delete topics["orders"][1]
    B->>B: if the topic has no subscriptions left, remove the topic
    B->>B: unlock mu
    B-->>C: done
    Note over B: Later publishes to "orders" reach only Subscription 2.<br/>Closing the channel safely while a publish may be<br/>sending to it is TBD
```

Before and after:

```text
Before: "orders" → { 1: Email, 2: Analytics }
After:  "orders" → { 2: Analytics }
```

## 7. Operation: Close

```mermaid
flowchart TD
    A["Close called"] --> B{"Already closed?"}
    B -- Yes --> C["Return, defined outcome (TBD)"]
    B -- No --> D["Mark broker closed. New Subscribe and Publish are rejected"]
    D --> E["Remove every subscription from the registry"]
    E --> F["Close each subscription's channel exactly once"]
    F --> G["Consumers' read loops see the closed channel and finish"]
```

## 8. Lifecycle of One Subscription

```mermaid
stateDiagram-v2
    [*] --> Active: Subscribe
    Active --> Active: Publish delivers a message
    Active --> Unsubscribed: Unsubscribe
    Active --> Closed: Broker Close
    Unsubscribed --> [*]
    Closed --> [*]
```

## 9. One Full Walk-Through

Using the same example as the problem statement:

| Step | Action | Registry after the step |
|---|---|---|
| 1 | Email subscribes to `orders` | `orders → {1: Email}` |
| 2 | Analytics subscribes to `orders` | `orders → {1: Email, 2: Analytics}` |
| 3 | Audit subscribes to `payments` | `orders → {1, 2}`, `payments → {3: Audit}` |
| 4 | Publish `{CREATED, "order-101"}` to `orders` | Email and Analytics each receive it; Audit receives nothing |
| 5 | Publish `{DELETED, "order-101"}` to `orders` | Same routing. The event type is inside the message, not the topic |
| 6 | Email unsubscribes | `orders → {2: Analytics}` |
| 7 | Publish `{CREATED, "order-102"}` to `orders` | Only Analytics receives it |
| 8 | Broker closes | Registry is empty; all channels are closed |

## 10. Which Diagram Answers Which Question

| Question | Section |
|---|---|
| What things exist and how are they related? | 1 and 2 |
| What happens when someone subscribes or publishes? | 3 and 4 |
| What if the consumer is slow? | 5 |
| What happens on unsubscribe and close? | 6, 7 and 8 |
| Can I trace a concrete example? | 9 |

In an interview, draw sections 2 (structure) and 4 (publish flow) first. They show the design in about two minutes.
