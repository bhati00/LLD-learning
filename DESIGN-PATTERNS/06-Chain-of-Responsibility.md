# Chain of Responsibility

> **Pass a request through an ordered sequence of handlers until it is handled or the chain ends.**

The chain separates a request producer from the specific handler that will process it. It is useful when handlers are ordered, optional, or assembled from configuration.

---

## The Core Idea

Each handler examines a request and either handles it, rejects it, or forwards it. The caller invokes the chain rather than selecting a concrete handler itself.

---

## Example: HTTP Middleware

Go's `http.Handler` makes middleware a familiar chain:

```go
package httpmiddleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request complete",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(started),
		)
	})
}

func RequireHeader(name string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(name) == "" {
			http.Error(w, "required header missing", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

The request travels through each wrapper until it reaches the final handler. A middleware can stop propagation by responding without calling `next`.

---

## When It Helps

- Several independent checks or transformations apply in a defined order.
- A handler may stop processing early, such as authorization rejecting a request.
- The chain should be assembled at startup and reused.

## Costs and Pitfalls

- Order is behavior: authentication, authorization, and logging may need deliberate placement.
- A handler that forgets to forward can silently terminate the chain.
- A handler that forwards more than once can duplicate work.
- Error propagation and observability need a consistent contract.
- If exactly one known implementation must always run, a direct call is clearer.

## Interview Check

Explain how the chain is composed, what a handler does to continue or stop, and how order affects correctness. Compare with Decorator: middleware is often a chain of same-interface wrappers, but its defining feature is sequential request handling and the ability to short-circuit.