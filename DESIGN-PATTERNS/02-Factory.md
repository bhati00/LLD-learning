# Factory Pattern

> **Move object-creation decisions behind a function or type that returns a useful abstraction.**

Factories help callers ask for behavior without knowing every concrete implementation detail. Go often needs only a constructor function; a larger factory type is justified when creation itself has meaningful state or dependencies.

---

## The Core Idea

Without a factory, every caller may need to know which concrete type to build, what configuration it requires, and how to validate that configuration. Centralizing those decisions reduces duplication and isolates change.

---

## Example: Select a Payment Processor

The caller depends on behavior, while the factory owns the selection rule:

```go
package payment

import "fmt"

type Processor interface {
	Charge(cents int64) error
}

type CardProcessor struct{ apiKey string }

func (p *CardProcessor) Charge(cents int64) error {
	return nil // provider call
}

type BankProcessor struct{ accountID string }

func (p *BankProcessor) Charge(cents int64) error {
	return nil // provider call
}

func NewProcessor(provider, credential string) (Processor, error) {
	switch provider {
	case "card":
		return &CardProcessor{apiKey: credential}, nil
	case "bank":
		return &BankProcessor{accountID: credential}, nil
	default:
		return nil, fmt.Errorf("unknown payment provider %q", provider)
	}
}
```

The selection switch is acceptable when it is the single, explicit creation boundary. A factory is not automatically better than a switch; the value is putting the decision in one place.

---

## Constructor Functions Are Factories Too

Most Go code needs no `Factory` interface:

```go
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}
```

Use an interface for the returned value when callers need interchangeable behavior, not just to make every constructor look abstract.

---

## When It Helps

- Creation requires validation, defaults, or multiple collaborators.
- Callers choose among implementations using configuration or input.
- You want to keep concrete implementation packages out of a higher-level package.

## Costs and Pitfalls

- A factory can become a large switchboard that knows every detail in the system.
- Adding a concrete type may still require modifying the factory; a factory alone does not guarantee the Open/Closed Principle.
- Avoid a factory interface with a single implementation and no realistic substitution need.
- Return errors for invalid configuration instead of panicking.

## Interview Check

Explain what creation complexity the factory removes, what abstraction it returns, and where new implementations are registered. Distinguish a simple constructor from Abstract Factory, which coordinates creation of related families of objects.