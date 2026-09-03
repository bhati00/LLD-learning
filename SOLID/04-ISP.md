# Interface Segregation Principle (ISP)

> **Clients should not be forced to depend on interfaces they do not use.**

An interface is a contract. ISP says: keep contracts small and focused.
A type should only need to implement the methods that are actually relevant to it.

---

## The Core Idea

A fat interface forces every implementor to deal with every method,
even the ones that don't apply. This creates noise, forces meaningless implementations,
and makes code fragile.

The fix: split the fat interface into smaller, focused ones. Each client
depends only on the slice it actually uses.

---

## Bad Example — Violation

A single `Animal` interface that every animal must implement:

```go
package zoo

type Animal interface {
    Eat()
    Sleep()
    Fly()
    Swim()
    Run()
}

// Dog can't fly — but the interface forces it to implement Fly().
type Dog struct{}

func (d Dog) Eat()   { fmt.Println("dog eats") }
func (d Dog) Sleep() { fmt.Println("dog sleeps") }
func (d Dog) Run()   { fmt.Println("dog runs") }
func (d Dog) Swim()  { fmt.Println("dog swims") }
func (d Dog) Fly()   {
    // Forced to implement something meaningless.
    // Common "fixes": panic, empty body, return error — all LSP violations.
    panic("dogs can't fly")
}

// Eagle can't swim (or swim well) — same problem.
type Eagle struct{}

func (e Eagle) Eat()   { fmt.Println("eagle eats") }
func (e Eagle) Sleep() { fmt.Println("eagle sleeps") }
func (e Eagle) Fly()   { fmt.Println("eagle flies") }
func (e Eagle) Run()   { fmt.Println("eagle runs") }
func (e Eagle) Swim()  {
    panic("eagles can't swim")
}
```

**What's wrong:**
- Every new animal must implement all 5 methods, even irrelevant ones
- Panicking implementations are LSP violations waiting to happen
- A caller using `Animal` can't safely call `Fly()` without knowing the concrete type
- Testing `Dog.Eat()` pulls in the complexity of the whole interface

---

## Good Example — Fixed

Break the fat interface into small, focused interfaces:

```go
package zoo

type Eater interface {
    Eat()
}

type Sleeper interface {
    Sleep()
}

type Flyer interface {
    Fly()
}

type Swimmer interface {
    Swim()
}

type Runner interface {
    Run()
}

// Dog implements only what a dog actually does.
type Dog struct{}

func (d Dog) Eat()   { fmt.Println("dog eats") }
func (d Dog) Sleep() { fmt.Println("dog sleeps") }
func (d Dog) Run()   { fmt.Println("dog runs") }
func (d Dog) Swim()  { fmt.Println("dog swims") }
// No Fly() — and that's fine.

// Eagle implements only what an eagle actually does.
type Eagle struct{}

func (e Eagle) Eat()   { fmt.Println("eagle eats") }
func (e Eagle) Sleep() { fmt.Println("eagle sleeps") }
func (e Eagle) Fly()   { fmt.Println("eagle flies") }
func (e Eagle) Run()   { fmt.Println("eagle runs") }
// No Swim() — and that's fine.
```

Now callers depend only on what they need:

```go
// This function only needs something that can fly — it doesn't care about the rest.
func MakeItFly(f Flyer) {
    f.Fly()
}

// This function only needs feeding behavior.
func FeedAll(animals []Eater) {
    for _, a := range animals {
        a.Eat()
    }
}
```

---

## Composing Interfaces When Needed

Sometimes you need a type that does several things. Compose interfaces instead
of writing a fat one:

```go
// Duck can do all three.
type Duck struct{}

func (d Duck) Eat()   { fmt.Println("duck eats") }
func (d Duck) Swim()  { fmt.Println("duck swims") }
func (d Duck) Fly()   { fmt.Println("duck flies") }

// A composite interface for the rare case you need all three.
type AquaticFlyer interface {
    Swimmer
    Flyer
}

func MigrateWaterfowl(birds []AquaticFlyer) {
    for _, b := range birds {
        b.Swim()
        b.Fly()
    }
}
```

The building blocks remain small. Composition creates larger contracts only where needed.

---

## Go's Standard Library Does This Perfectly

The `io` package is the best real-world example of ISP in Go:

```go
// Each interface has exactly one method.
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}

// Composed only when needed.
type ReadWriter interface {
    Reader
    Writer
}

type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

`os.File` implements `ReadWriteCloser`. A function that only needs to read
accepts `io.Reader` — it doesn't care if the source is a file, a network
connection, a buffer, or a test stub. This is ISP making the entire ecosystem composable.

---

## Real-World Scenario: Notification Service

You're building a notification system that sends alerts via different channels.

**Violates ISP:**
```go
type Notifier interface {
    SendEmail(to, subject, body string) error
    SendSMS(to, message string) error
    SendPush(deviceToken, title, body string) error
    SendSlack(channel, message string) error
}

// EmailNotifier is forced to implement SMS, Push, Slack.
type EmailNotifier struct{}

func (e EmailNotifier) SendEmail(to, subject, body string) error { /* real impl */ return nil }
func (e EmailNotifier) SendSMS(to, message string) error         { return errors.New("not supported") }
func (e EmailNotifier) SendPush(token, title, body string) error { return errors.New("not supported") }
func (e EmailNotifier) SendSlack(channel, msg string) error      { return errors.New("not supported") }
```

**Satisfies ISP:**
```go
type EmailSender interface {
    SendEmail(to, subject, body string) error
}

type SMSSender interface {
    SendSMS(to, message string) error
}

type PushSender interface {
    SendPush(deviceToken, title, body string) error
}

type SlackSender interface {
    SendSlack(channel, message string) error
}

// EmailNotifier only implements what it truly supports.
type EmailNotifier struct{ apiKey string }

func (e EmailNotifier) SendEmail(to, subject, body string) error {
    // real SMTP/API logic
    return nil
}

// The alert dispatcher only asks for what it needs.
type AlertDispatcher struct {
    email EmailSender
    sms   SMSSender
}

func (d *AlertDispatcher) Alert(userEmail, userPhone, msg string) {
    d.email.SendEmail(userEmail, "Alert", msg)
    d.sms.SendSMS(userPhone, msg)
}
```

---

## How to Spot an ISP Violation

- An interface has more than 3–5 methods and they don't form a single cohesive concept
- Implementations have `panic("not supported")` or `return nil // no-op` bodies
- A test doubles a large interface but only cares about 1–2 methods
- You grep for the interface name and find that most usages only call a subset of its methods

---

## Go-Specific Notes

**Define interfaces at the consumer, not the producer.** In Go, interfaces are
typically defined where they are *used*, not where they are *implemented*.
This naturally creates small, purpose-specific interfaces.

```go
// BAD: package user defines a big interface that other packages must satisfy.
// GOOD: package notification defines a tiny EmailSender interface it needs,
//       and user.EmailNotifier satisfies it implicitly.
```

**Prefer 1–3 method interfaces.** If your interface has more than 3–4 methods,
ask whether it's really one concept or two concepts that should be split.

**Mock pain reveals ISP violations.** If writing a mock for a test requires you
to stub 8 methods when you only care about 1, the interface is too fat.
