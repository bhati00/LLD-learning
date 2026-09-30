# Singleton Pattern

> **Ensure a type has one shared instance and provide a single way to access it.**

In Go, a package-level value is often simpler than a Singleton class. Use the pattern only when the application truly needs one process-wide instance, not merely because constructing a value is inconvenient.

---

## The Core Idea

A Singleton combines two decisions:

- **Instance control:** initialization happens once.
- **Global access:** callers can reach that instance from a known place.

The second decision is the risky one. Global access hides dependencies and makes tests order-dependent. Prefer explicit dependency injection unless shared identity is a real requirement.

---

## Go Example: Lazy, Thread-Safe Initialization

`sync.Once` makes initialization safe when multiple goroutines call `Get` at the same time.

```go
package config

import "sync"

type Registry struct {
	entries map[string]string
}

var (
	registryOnce sync.Once
	registry     *Registry
)

func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{entries: make(map[string]string)}
	})
	return registry
}
```

Every caller receives the same pointer. `sync.Once` also safely publishes the initialized value to concurrent callers.

---

## A Simpler Alternative: Package Initialization

When lazy initialization is unnecessary, initialize a package-level value directly:

```go
var defaultRegistry = &Registry{entries: make(map[string]string)}

func DefaultRegistry() *Registry {
	return defaultRegistry
}
```

Go initializes package variables before package use. This avoids synchronization and is often the clearer choice for cheap, predictable setup.

---

## When It Helps

- One process-wide metrics registry or immutable configuration snapshot is required.
- Initialization is expensive and should happen at most once.
- Multiple callers must share identity or coordinated state.

## Costs and Pitfalls

- A global accessor hides the dependency from constructors and function signatures.
- Shared mutable state creates races unless its methods synchronize access.
- Tests can leak state into one another; resetting a `sync.Once` is not supported.
- A Singleton is process-local, not a distributed singleton across servers.
- Do not use it to avoid passing a dependency that belongs in a struct.

For most services, create the dependency in `main` and inject it into the components that need it. That preserves one instance without making it globally accessible.

---

## Interview Check

Be ready to explain the difference between **one instance** and **global access**, how concurrent initialization is protected, and how tests can avoid shared mutable state. Mention `sync.Once` for one-time initialization, but do not claim it makes all future operations on the instance thread-safe.