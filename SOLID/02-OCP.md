# Open/Closed Principle (OCP)

> **Software entities should be open for extension, but closed for modification.**

"Open for extension" means you can add new behavior.
"Closed for modification" means adding new behavior does NOT require changing existing, tested code.

---

## The Core Idea

When requirements change, you should be able to **add** new code rather than **edit** old code.
Editing old code risks breaking things that already work. Adding new code in isolation is safer.

The mechanism that makes this possible: **abstractions (interfaces)**.

---

## Bad Example — Violation

A discount calculator that needs to know about every discount type:

```go
package pricing

type DiscountType string

const (
    SeasonalDiscount  DiscountType = "seasonal"
    LoyaltyDiscount   DiscountType = "loyalty"
    StudentDiscount   DiscountType = "student"
)

// Every time a new discount type is added, this function must be modified.
func ApplyDiscount(price float64, discountType DiscountType) float64 {
    switch discountType {
    case SeasonalDiscount:
        return price * 0.90
    case LoyaltyDiscount:
        return price * 0.85
    case StudentDiscount:
        return price * 0.80
    default:
        return price
    }
}
```

**What's wrong:**
- Adding a "Flash Sale" discount requires opening and editing `ApplyDiscount`
- The `switch` grows forever as the business adds more discount types
- Every edit risks accidentally breaking an existing discount calculation
- The function must be redeployed and re-tested even when the change is unrelated to it

---

## Good Example — Fixed

Define a `Discounter` interface. Each discount type is its own struct.
The `ApplyDiscount` function never changes — it just calls the interface.

```go
package pricing

// The abstraction — this never changes.
type Discounter interface {
    Apply(price float64) float64
}

// Each implementation is isolated.

type SeasonalDiscount struct{}

func (d SeasonalDiscount) Apply(price float64) float64 {
    return price * 0.90
}

type LoyaltyDiscount struct{}

func (d LoyaltyDiscount) Apply(price float64) float64 {
    return price * 0.85
}

type StudentDiscount struct{}

func (d StudentDiscount) Apply(price float64) float64 {
    return price * 0.80
}

// This function is now closed for modification.
func ApplyDiscount(price float64, d Discounter) float64 {
    return d.Apply(price)
}
```

Adding a Flash Sale discount now looks like this — zero changes to existing code:

```go
// New file: flash_sale_discount.go

type FlashSaleDiscount struct {
    Percent float64
}

func (d FlashSaleDiscount) Apply(price float64) float64 {
    return price * (1 - d.Percent/100)
}
```

---

## Stacking Discounts (Extension without Modification)

OCP enables composable behavior. Here's a discount chain that never touches existing types:

```go
// CompositeDiscount applies multiple discounts in sequence.
type CompositeDiscount struct {
    discounts []Discounter
}

func (c CompositeDiscount) Apply(price float64) float64 {
    for _, d := range c.discounts {
        price = d.Apply(price)
    }
    return price
}

// Usage — no existing code was modified.
combo := CompositeDiscount{
    discounts: []Discounter{
        SeasonalDiscount{},
        LoyaltyDiscount{},
    },
}
finalPrice := ApplyDiscount(100.0, combo) // 100 * 0.90 * 0.85 = 76.50
```

This is the **Strategy pattern** — OCP is the principle, Strategy is the pattern that implements it.

---

## Real-World Scenario: Rate Limiter

Your rate limiter supports token bucket today. Tomorrow the team wants sliding window.

**Violates OCP:**
```go
func (rl *RateLimiter) Allow(key string, algo string) bool {
    if algo == "token_bucket" {
        // token bucket logic
    } else if algo == "sliding_window" {
        // sliding window logic
    }
    return false
}
```

**Satisfies OCP:**
```go
type Algorithm interface {
    Allow(key string) bool
}

type RateLimiter struct {
    algo Algorithm
}

func (rl *RateLimiter) Allow(key string) bool {
    return rl.algo.Allow(key)
}

// New algorithm = new file, zero edits to RateLimiter.
type SlidingWindowAlgorithm struct { /* ... */ }
func (s *SlidingWindowAlgorithm) Allow(key string) bool { /* ... */ return true }
```

---

## How to Spot an OCP Violation

Look for these patterns in code:
- `switch` on a type/kind field that grows over time
- `if algo == "x" ... else if algo == "y"` branching
- Comments like `// add new type here`
- Functions that import types they shouldn't need to know about

---

## Go-Specific Notes

**Interfaces are the tool.** Go's implicit interfaces make OCP very natural.
You don't need abstract classes or inheritance.

**Don't abstract prematurely.** OCP does NOT mean "wrap everything in an interface."
Extract an interface when you see the second use case, not the first.
The rule of thumb: **abstract at the point of variation**, not everywhere.

**Functions are values too.** For simpler cases, a `func` type is a valid abstraction:

```go
type DiscountFunc func(price float64) float64

func ApplyDiscount(price float64, fn DiscountFunc) float64 {
    return fn(price)
}

// Usage
seasonal := func(p float64) float64 { return p * 0.90 }
ApplyDiscount(100.0, seasonal)
```

Use interface when the behavior has state; use a function type when it's stateless.
