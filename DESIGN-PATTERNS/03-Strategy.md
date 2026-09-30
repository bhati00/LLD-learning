# Strategy Pattern

> **Put interchangeable algorithms behind a common contract and choose one at runtime.**

Strategy replaces growing conditional logic with composed behavior. It is a direct way to apply the Open/Closed Principle when a behavior varies independently from the code that uses it.

---

## The Core Idea

The context owns the workflow; a strategy owns one algorithm within that workflow. The context should not need to know how the algorithm works.

---

## Example: Shipping Cost

```go
package shipping

type Quote struct {
	WeightGrams int
	DistanceKM  int
}

type RateStrategy interface {
	Cost(Quote) int64 // cents
}

type StandardRate struct{}

func (StandardRate) Cost(q Quote) int64 {
	return int64(q.WeightGrams/100 + q.DistanceKM/10)
}

type ExpressRate struct{}

func (ExpressRate) Cost(q Quote) int64 {
	return int64(500 + q.WeightGrams/50 + q.DistanceKM/5)
}

type Quoter struct {
	rate RateStrategy
}

func NewQuoter(rate RateStrategy) Quoter {
	return Quoter{rate: rate}
}

func (q Quoter) Cost(input Quote) int64 {
	return q.rate.Cost(input)
}
```

Selection happens at the composition boundary:

```go
quoter := NewQuoter(ExpressRate{})
cost := quoter.Cost(Quote{WeightGrams: 1200, DistanceKM: 80})
```

---

## Function as Strategy

For a small, stateless behavior, a function type can be more idiomatic than an interface:

```go
type RateFunc func(Quote) int64

func (f RateFunc) Cost(q Quote) int64 { return f(q) }
```

Use a named interface when implementations have state, multiple methods, or deserve distinct types. Use a function when the behavior is one operation and does not need its own object identity.

---

## When It Helps

- Several algorithms implement the same business operation.
- The algorithm is selected by configuration, request, or runtime context.
- Tests need to supply a deterministic alternative.

## Costs and Pitfalls

- Each strategy adds a type and a selection/wiring decision.
- Do not create a strategy for a conditional that is stable, tiny, and unlikely to vary.
- Keep shared workflow in the context; do not duplicate it in each strategy.
- Validate the selected strategy at startup rather than accepting an unusable nil dependency.

## Interview Check

Identify the context, strategy contract, concrete algorithms, and selection point. Compare Strategy with State: Strategy selects an algorithm, while State changes an object's behavior as its internal state transitions.