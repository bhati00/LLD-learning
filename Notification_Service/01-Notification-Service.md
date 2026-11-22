# Notification Service: Requirements

[Back to LLD topics](../LLD-Topics.md)

**Goal:** brainstorm the design yourself, then implement it in Go. This doc only lists *what* to build, not *how*.

---

## 1. Problem Statement

Build a service that sends notifications to users over multiple channels (Email, SMS, Push, and so on). It must be easy to extend, reliable under provider failures, and must respect user preferences.

---

## 2. Functional Requirements (FR)

### Core
- [ ] **FR1:** Send a notification to a user on a given channel (Email, SMS, Push).
- [ ] **FR2:** Send one notification to multiple channels at once.
- [ ] **FR3:** Adding a new channel (e.g. Slack, WhatsApp) must not require modifying existing code.
- [ ] **FR4:** Each notification has a type (OTP, Transactional, Marketing, Alert) and a priority (High, Medium, Low).

### Templates *(optional, see Section 9)*
- [ ] **FR5 (optional):** Support message templates with variables, e.g. `Hi {{name}}, your OTP is {{otp}}`.
- [ ] **FR6 (optional):** A template can have a different body per channel (short for SMS, rich for Email).

### User preferences
- [ ] **FR7:** A user can opt in or out per channel (e.g. no SMS).
- [ ] **FR8:** A user can opt in or out per notification type (e.g. no Marketing).
- [ ] **FR9:** Critical types (OTP, security alerts) must bypass opt-out.
- [ ] **FR10:** Quiet hours: non-critical notifications are held or dropped during them *(stretch)*.

### Delivery
- [ ] **FR11:** Failed sends are retried with backoff, up to a max attempt count.
- [ ] **FR12:** If a channel keeps failing, fall back to another channel (e.g. Push fails, so send SMS).
- [ ] **FR13:** Higher-priority notifications are delivered before lower-priority ones.
- [ ] **FR14:** Support scheduled notifications (send at time T) *(stretch)*.

### Tracking
- [ ] **FR15:** Track each notification's status: `Pending → Queued → Sent → Delivered / Failed`.
- [ ] **FR16:** Query the status and history of a notification by ID.

---

## 3. Non-Functional Requirements (NFR)

### Performance and scalability
- [ ] **NFR1:** `Send()` must be **non-blocking** for the caller. A slow provider must not slow the API.
- [ ] **NFR2:** Bounded concurrency: a fixed number of workers, not unlimited goroutines.
- [ ] **NFR3:** Bounded queue with a defined **backpressure** behavior when full (block, reject, or drop low priority).

### Reliability
- [ ] **NFR4:** **Idempotency:** the same request (same idempotency key) must never send twice.
- [ ] **NFR5:** One failing channel or provider must not affect the others (**isolation**).
- [ ] **NFR6:** **Circuit breaker** per provider, to stop hammering a dead one.
- [ ] **NFR7:** **Rate limit** per user (anti-spam) and per provider (respect provider quotas).
- [ ] **NFR8:** **Graceful shutdown:** stop accepting new work, drain in-flight work, and honor `context` cancellation.
- [ ] **NFR9:** Delivery guarantee: decide **at-least-once** or **at-most-once** and justify it.

### Design quality
- [ ] **NFR10:** **Extensible** (OCP): new channels, templates, or filters don't change the core.
- [ ] **NFR11:** **Testable:** all providers are behind interfaces, so tests use fakes with no real network.
- [ ] **NFR12:** **Thread-safe:** safe under concurrent callers (`go test -race` passes).
- [ ] **NFR13:** **Configurable** (workers, retries, queue size) via functional options.
- [ ] **NFR14:** **Observable:** log or count sent/failed/retried per channel.

---

## 4. Out of Scope

- Real provider integrations (SMTP, Twilio, FCM). Use fake senders that sleep or randomly fail.
- Persistent storage. In-memory is fine.
- Auth, UI, and an HTTP API (optional at the end).

---

## 5. Questions to Brainstorm

Answer these *before* writing code.

**Modeling**
1. What are the core entities? (Notification, Channel, Template, User, Preference, ...)
2. What does the `Channel` interface look like? What is the smallest useful one?
3. Who decides which channels a notification goes to: the caller, or the service based on preferences?

**Flow**
4. Draw the pipeline from `Send()` to delivered. Which steps happen *before* the queue and which *after*?
5. Where do the preference check, dedup, and rate limit sit, and in what order? Why that order?
6. Is the preference check synchronous or in the worker? What are the trade-offs?

**Concurrency**
7. One queue or a queue per priority? How do you stop Low starving behind High?
8. What happens when the queue is full?
9. How do you shut down without losing queued work?

**Reliability**
10. Which errors are retryable (timeout) and which are not (invalid phone number)?
11. How do retry and circuit breaker interact? Does retry wrap the breaker or the reverse?
12. How do you generate the idempotency key, and how long do you remember it?
13. What if a send succeeds but the status update fails?

**Extensibility**
14. How do you add a Slack channel? Count the files you touch. The answer should be 1 or 2.
15. How would you swap in-memory storage for a DB later?

---

## 6. Suggested Build Order

1. **Core:** `Notification`, `Channel` interface, Email/SMS/Push fakes, registry or factory, synchronous send.
2. **Preferences:** a preference filter (templates are optional, see Section 9).
3. **Async:** queue plus worker pool, priority handling, and graceful shutdown.
4. **Reliability:** retry with backoff, fallback, idempotency.
5. **Resilience:** circuit breaker and rate limiter (reuse your existing ones).
6. **Tracking and tests:** status store, fake channels, `go test -race`.

---

## 7. Patterns to Consider (no spoilers)

You've already covered most of these. Decide where each one fits:

Strategy, Factory, Observer, Decorator, Chain of Responsibility, State, Functional Options, Worker Pool, Circuit Breaker, Rate Limiter.

---

## 8. Done Checklist

- [ ] A new channel was added by touching at most 2 files.
- [ ] Opted-out users never receive non-critical messages.
- [ ] Killing a provider (always fails) doesn't block or crash other channels.
- [ ] The same idempotency key sent twice delivers once.
- [ ] `go test -race ./...` passes.
- [ ] Shutdown drains the queue without goroutine leaks.
- [ ] I can explain every design decision and its trade-off out loud.

---

## 9. Optional: Templates (FR5, FR6)

Too deep for a typical 45-60 minute interview. In the interview, `Notification` carries a plain `subject` and `body`, and you say in one sentence: "Templates would be a per-channel body with variable substitution, behind a small `Renderer` interface." Build this only after the core is done.

**Model**

| Type | Purpose |
|---|---|
| `ChannelTemplate` | Raw text for one channel: `subject` (email only) and `body`. |
| `Template` | An `id` plus `variants map[ChannelType]ChannelTemplate`: one logical message, one variant per channel. |
| `Content` | The rendered output (`subject`, `body`), passed to `Channel.Send`. |

`Notification` would then hold `templateID string` and `data map[string]any` instead of `subject` and `body`. The same `data` is used by every channel variant.

**Example**
```go
Template{
    id: "otp",
    variants: map[ChannelType]ChannelTemplate{
        ChannelSMS:   {body: "Your OTP is {{.otp}}"},
        ChannelEmail: {subject: "Verify login", body: "Hi {{.name}}, your OTP is {{.otp}}"},
    },
}
```

**Flow:** `Service.Send` loads the template by ID, calls `Render(channel, data)` (use `text/template`), and passes the resulting `Content` to the channel. Return an error if the template has no variant for that channel.

**Questions to brainstorm**
1. What happens when a variant is missing for a channel: error, skip, or fall back to a default?
2. What happens when `data` is missing a variable the template uses?
3. Where does the template store live, and how does it stay swappable?
4. Should the channel format the text (SMS truncation, Slack blocks) or only the template?
