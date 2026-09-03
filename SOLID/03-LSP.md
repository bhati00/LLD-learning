# Liskov Substitution Principle (LSP)

> **If S is a subtype of T, then objects of type T may be replaced with objects of type S
> without altering the correctness of the program.**

In plain terms: **if a function accepts an interface, every implementation of that
interface must behave in a way the function can rely on.**

---

## The Core Idea

LSP is about **behavioral contracts**, not just method signatures.

An interface defines a contract: "I promise to do X." Every type that satisfies
the interface must honor that promise fully. If a type implements the interface
but secretly panics, returns wrong results, or ignores the call, the contract is broken —
and code that depends on the interface becomes unpredictable.

---

## The Classic Violation: Rectangle and Square

This is the canonical LSP example. In math, a Square IS-A Rectangle.
In software, making Square implement Rectangle causes problems.

```go
package shapes

type Rectangle struct {
    width, height float64
}

func (r *Rectangle) SetWidth(w float64)  { r.width = w }
func (r *Rectangle) SetHeight(h float64) { r.height = h }
func (r *Rectangle) Area() float64       { return r.width * r.height }

// Square "extends" Rectangle by overriding setters.
type Square struct {
    Rectangle
}

// Square enforces equal sides — but this breaks Rectangle's contract.
func (s *Square) SetWidth(w float64) {
    s.width = w
    s.height = w // silently changes height too
}

func (s *Square) SetHeight(h float64) {
    s.height = h
    s.width = h // silently changes width too
}
```

Now this function breaks when given a Square:

```go
// This function assumes Rectangle contract: width and height are independent.
func assertArea(r *Rectangle, w, h float64) {
    r.SetWidth(w)
    r.SetHeight(h)
    expected := w * h
    actual := r.Area()
    if actual != expected {
        fmt.Printf("FAIL: expected %.0f, got %.0f\n", expected, actual)
    }
}

rect := &Rectangle{}
assertArea(rect, 4, 5) // PASS: 20

sq := &Square{}
assertArea(&sq.Rectangle, 4, 5) // FAIL: expected 20, got 25
// SetHeight(5) also set width to 5, so area is 5*5=25
```

The function assumed: "setting width doesn't affect height." Square broke that assumption.
LSP is violated.

**Fix:** Don't model IS-A through embedding when the behavioral contracts diverge.
Keep them separate types, both implementing a common `Shape` interface if needed.

```go
type Shape interface {
    Area() float64
}

type Rectangle struct { width, height float64 }
func (r Rectangle) Area() float64 { return r.width * r.height }

type Square struct { side float64 }
func (s Square) Area() float64 { return s.side * s.side }

// Both satisfy Shape. No substitution problem.
func printArea(s Shape) {
    fmt.Printf("Area: %.2f\n", s.Area())
}
```

---

## Go-Specific Violation: Panic or No-op in Interface Implementation

In Go, LSP violations most commonly look like this:

```go
type Storage interface {
    Read(key string) ([]byte, error)
    Write(key string, data []byte) error
    Delete(key string) error
}

// ReadOnlyStorage satisfies the interface signature but violates the contract.
type ReadOnlyStorage struct {
    data map[string][]byte
}

func (s *ReadOnlyStorage) Read(key string) ([]byte, error) {
    return s.data[key], nil
}

func (s *ReadOnlyStorage) Write(key string, data []byte) error {
    // Violates LSP: caller expects Write to work, but it silently does nothing.
    return nil
}

func (s *ReadOnlyStorage) Delete(key string) error {
    // Violates LSP: caller expects Delete to work, but it panics.
    panic("read-only storage: delete not supported")
}
```

Any code using the `Storage` interface cannot safely use `ReadOnlyStorage` in its place.

**Fix:** Use the Interface Segregation Principle (ISP) — split the interface:

```go
type Reader interface {
    Read(key string) ([]byte, error)
}

type Writer interface {
    Write(key string, data []byte) error
}

type Deleter interface {
    Delete(key string) error
}

// ReadWriteDeleter is the full contract for types that support everything.
type ReadWriteDeleter interface {
    Reader
    Writer
    Deleter
}

// ReadOnlyStorage only implements what it can truly honor.
type ReadOnlyStorage struct {
    data map[string][]byte
}

func (s *ReadOnlyStorage) Read(key string) ([]byte, error) {
    return s.data[key], nil
}
// No Write or Delete — and no violation.
```

This shows **LSP and ISP are closely related**: narrow interfaces prevent LSP violations
by not demanding contracts a type cannot honestly keep.

---

## The Behavioral Contract Rules

When designing or implementing an interface, an implementation must:

| Rule | What it means |
|---|---|
| **Preconditions** | Must not require *stronger* preconditions than the interface promises |
| **Postconditions** | Must guarantee *at least* what the interface promises |
| **Invariants** | Must preserve any invariant the interface establishes |
| **Exceptions** | Must not raise new, unexpected error types that callers don't know to handle |

---

## Practical Check

Before implementing an interface method, ask:

> "If a caller uses this interface and my type, will they observe any surprise?"

Surprises that violate LSP:
- A method that should return data returns `nil` without an error
- A method that should mutate state silently does nothing
- A method that should be idempotent changes state multiple times
- A method panics when the interface contract doesn't mention that possibility

---

## Go-Specific Notes

**Embedding is NOT inheritance.** Embedding in Go promotes methods, but the embedded
type's behavior is not overridden in the Go sense — methods on the outer type shadow them.
Be careful: if you embed a type to "inherit" behavior and then partially override it,
you may create LSP violations like the Rectangle/Square example above.

**`error` return values are part of the contract.** An interface method returning `error`
implies callers will check it. Returning a nil error when the operation failed (silently)
is an LSP violation in spirit.

**Test LSP by writing tests against the interface, not the concrete type:**

```go
func testStorage(t *testing.T, s Storage) {
    err := s.Write("k", []byte("v"))
    if err != nil {
        t.Fatalf("Write failed: %v", err)
    }
    data, err := s.Read("k")
    if err != nil || string(data) != "v" {
        t.Fatalf("Read after Write failed")
    }
}
```

Run this test with every concrete `Storage` implementation. If any fail, LSP is violated.
