# Decorator Pattern

> **Wrap an object with another object that adds behavior while preserving the same interface.**

Decorator composes cross-cutting behavior around a core implementation. Because the wrapper implements the same contract, callers can use the decorated and undecorated values interchangeably.

---

## The Core Idea

Instead of adding logging, metrics, retries, or caching directly to a concrete type, place each behavior in a wrapper. Wrappers can be composed in different combinations without creating a subclass for every combination.

---

## Example: Add Logging to a Sender

```go
package notify

import "log/slog"

type Sender interface {
	Send(to, message string) error
}

type LoggingSender struct {
	next   Sender
	logger *slog.Logger
}

func (s LoggingSender) Send(to, message string) error {
	s.logger.Info("sending notification", "to", to)
	if err := s.next.Send(to, message); err != nil {
		s.logger.Error("notification failed", "error", err)
		return err
	}
	return nil
}
```

Compose at startup:

```go
var sender Sender = providerSender
sender = LoggingSender{next: sender, logger: logger}
```

The outer wrapper receives the call first and delegates inward. Wrapping order matters when multiple decorators are used; for example, logging outside a retry wrapper logs one request, while logging inside it may log every attempt.

---

## When It Helps

- Add metrics, tracing, logging, authorization, caching, or retry behavior around an existing contract.
- Combine optional behaviors without multiplying concrete types.
- Keep infrastructure concerns out of core business implementations.

## Costs and Pitfalls

- Deep wrapper stacks make call flow and debugging less obvious.
- Behavior depends on wrapper order.
- A wrapper must preserve the interface contract, including errors and context cancellation.
- Retrying non-idempotent operations can duplicate side effects.
- Wrapping types that expose extra methods may hide those methods unless the design handles them explicitly.

## Interview Check

Show that both the component and wrapper satisfy the same interface, explain delegation order, and identify how errors and context pass through the chain. Compare Decorator (same interface, added behavior) with Adapter (changes an incompatible interface).