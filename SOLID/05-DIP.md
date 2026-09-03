# Dependency Inversion Principle (DIP)

> **High-level modules should not depend on low-level modules. Both should depend on abstractions.**
> **Abstractions should not depend on details. Details should depend on abstractions.**

In plain terms: your business logic should not be wired to specific implementations.
It should talk to interfaces. The concrete implementations are plugged in from outside.

---

## The Core Idea

Think of it as a power outlet analogy:
- Your laptop (high-level) doesn't care whether it's plugged into a UK, US, or EU socket
- It talks to the "power supply" interface (abstraction)
- The adapter (low-level detail) handles the specifics

Without DIP, your business logic is hardwired to a specific database, HTTP client,
or logger. Changing any of them means cracking open your core logic.

With DIP, the core logic owns the interface contract. The concrete stuff is injected in.

---

## Bad Example — Violation

An `OrderService` that directly creates and uses a concrete MySQL repository:

```go
package order

import "database/sql"

// Low-level module — the concrete DB implementation.
type MySQLOrderRepository struct {
    db *sql.DB
}

func (r *MySQLOrderRepository) Save(o Order) error {
    _, err := r.db.Exec("INSERT INTO orders ...")
    return err
}

// High-level module — directly instantiates the low-level module.
type OrderService struct {
    repo *MySQLOrderRepository // hardwired to MySQL
}

func NewOrderService(db *sql.DB) *OrderService {
    return &OrderService{
        repo: &MySQLOrderRepository{db: db}, // created internally
    }
}

func (s *OrderService) PlaceOrder(o Order) error {
    return s.repo.Save(o)
}
```

**What's wrong:**
- `OrderService` is coupled to MySQL. Switching to PostgreSQL or DynamoDB means
  rewriting `OrderService` too.
- Testing `PlaceOrder` requires a live MySQL connection — there's no way to substitute
  a fake repository.
- `OrderService` knows how `MySQLOrderRepository` is constructed — it owns a dependency
  it shouldn't care about.

---

## Good Example — Fixed

Define the interface that the high-level module needs. Inject the dependency from outside.

```go
package order

// The abstraction — defined by the consumer (OrderService), not the producer.
type OrderRepository interface {
    Save(o Order) error
    FindByID(id string) (Order, error)
}

// High-level module depends on the interface, not the concrete type.
type OrderService struct {
    repo OrderRepository // abstraction
}

// The concrete implementation is injected — OrderService doesn't create it.
func NewOrderService(repo OrderRepository) *OrderService {
    return &OrderService{repo: repo}
}

func (s *OrderService) PlaceOrder(o Order) error {
    return s.repo.Save(o)
}
```

Now the concrete implementations live in a separate layer:

```go
package mysql

import "database/sql"

// Low-level module — detail depends on the abstraction (it satisfies OrderRepository).
type MySQLOrderRepository struct {
    db *sql.DB
}

func (r *MySQLOrderRepository) Save(o order.Order) error {
    _, err := r.db.Exec("INSERT INTO orders ...")
    return err
}

func (r *MySQLOrderRepository) FindByID(id string) (order.Order, error) {
    // query logic
    return order.Order{}, nil
}
```

And in tests, a fake is trivial:

```go
package order_test

type fakeRepo struct {
    saved []Order
}

func (f *fakeRepo) Save(o Order) error {
    f.saved = append(f.saved, o)
    return nil
}

func (f *fakeRepo) FindByID(id string) (Order, error) {
    return Order{}, nil
}

func TestPlaceOrder(t *testing.T) {
    repo := &fakeRepo{}
    svc := NewOrderService(repo)

    err := svc.PlaceOrder(Order{ID: "1"})
    if err != nil {
        t.Fatal(err)
    }
    if len(repo.saved) != 1 {
        t.Fatal("expected one saved order")
    }
}
```

Zero real database needed. Business logic is tested in isolation.

---

## Dependency Injection Patterns in Go

DIP requires injecting dependencies. Go has three common ways to do this.

### 1. Constructor Injection (most common)

```go
type NotificationService struct {
    emailer EmailSender
    logger  Logger
}

func NewNotificationService(emailer EmailSender, logger Logger) *NotificationService {
    return &NotificationService{emailer: emailer, logger: logger}
}
```

Use this when the dependency is required for the type to function at all.

### 2. Method Injection

```go
func (s *ReportService) Generate(w io.Writer, fetcher DataFetcher) error {
    data, err := fetcher.Fetch()
    if err != nil {
        return err
    }
    return s.render(w, data)
}
```

Use this when the dependency varies per call, not per instance.

### 3. Struct Field (explicit wiring)

```go
type App struct {
    OrderSvc  *OrderService
    UserSvc   *UserService
    DB        *sql.DB
}

func main() {
    db := connectDB()
    repo := &mysql.MySQLOrderRepository{DB: db}
    app := App{
        OrderSvc: order.NewOrderService(repo),
    }
    // ...
}
```

`main()` (or an initialization layer) is the one place where everything is wired together.
Business logic never imports infrastructure packages — only `main` does.

---

## The Dependency Graph Should Point Inward

A well-structured Go project using DIP looks like this:

```
main.go            — wires everything together, imports all packages
  └── service/     — business logic, imports only domain + interfaces
        └── domain/    — pure domain types (Order, User), no imports
  └── repository/  — DB implementations, imports domain
  └── http/        — HTTP handlers, imports service interfaces
```

The arrows of import should always point **toward** the domain/core.
`service` never imports `repository`. `repository` imports `domain`.
`main` imports everything and wires it up.

This is also called **Clean Architecture** or **Hexagonal Architecture** — DIP is the
principle that enables it.

---

## Real-World Scenario: Logger

**Violates DIP:**
```go
import "go.uber.org/zap"

type PaymentService struct {
    logger *zap.Logger // hardwired to zap
}
```

If you decide to switch to slog or a custom logger, you must change `PaymentService`.
More critically, unit tests now need a real zap logger.

**Satisfies DIP:**
```go
// Your own minimal interface — defined in the package that needs it.
type Logger interface {
    Info(msg string, fields ...any)
    Error(msg string, fields ...any)
}

type PaymentService struct {
    logger Logger // abstraction
}

func NewPaymentService(logger Logger) *PaymentService {
    return &PaymentService{logger: logger}
}
```

`*zap.Logger` and `*slog.Logger` both satisfy this interface. In tests, use a no-op logger.

---

## How to Spot a DIP Violation

- A struct field is a concrete type (`*sql.DB`, `*http.Client`, `*zap.Logger`) instead of an interface
- `NewXxx()` constructors instantiate their own dependencies internally
- Business logic packages import infrastructure packages (e.g., `service` imports `mysql`)
- "I can't test this without a real database / real HTTP server"

---

## Go-Specific Notes

**Define the interface in the package that uses it, not the package that implements it.**
This is the opposite of Java conventions. It keeps the consumer in control of the contract.

```go
// GOOD: service package defines what it needs.
package service

type UserStore interface {
    FindByEmail(email string) (User, error)
}

// GOOD: db package implements it — no import of service needed.
package db

type PostgresUserStore struct { /* ... */ }
func (s *PostgresUserStore) FindByEmail(email string) (service.User, error) { /* ... */ }
```

**`io.Reader` and `io.Writer` are DIP in action.** Functions in the standard library
accept these interfaces so they work with files, buffers, network connections, or test
readers without depending on any of them concretely.

**Avoid global state.** Package-level variables like `var db *sql.DB` are a form of
hidden dependency — callers can't control what they get. Prefer passing dependencies
explicitly through constructors or function arguments.
