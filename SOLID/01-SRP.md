# Single Responsibility Principle (SRP)

> **A type should have only one reason to change.**

"Reason to change" means: who or what would force you to modify this code?
If the answer is "the business team AND the infrastructure team AND the security team",
the type has too many responsibilities.

---

## The Core Idea

SRP is not about doing only one thing. It's about **owning one concern**.

A `UserService` that handles:
- database queries
- password hashing
- sending welcome emails

...has three reasons to change. If the email provider changes, you touch `UserService`.
If the hashing algorithm changes, you touch `UserService`. If the DB schema changes,
you touch `UserService`. These are independent concerns and should live separately.

---

## Bad Example — Violation

```go
package user

import (
    "database/sql"
    "fmt"
    "net/smtp"
    "crypto/sha256"
)

type UserService struct {
    db *sql.DB
}

// Responsibility 1: database
func (s *UserService) Save(u User) error {
    _, err := s.db.Exec("INSERT INTO users (email, password) VALUES (?, ?)", u.Email, u.Password)
    return err
}

// Responsibility 2: password hashing (security concern)
func (s *UserService) HashPassword(password string) string {
    h := sha256.Sum256([]byte(password))
    return fmt.Sprintf("%x", h)
}

// Responsibility 3: email sending (infrastructure concern)
func (s *UserService) SendWelcomeEmail(email string) error {
    return smtp.SendMail(
        "smtp.example.com:587",
        nil,
        "no-reply@example.com",
        []string{email},
        []byte("Welcome!"),
    )
}
```

**What's wrong:**
- A change in SMTP config forces you to open `UserService`
- A change in hashing algorithm forces you to open `UserService`
- A DB schema change forces you to open `UserService`
- Testing `Save` requires a real DB even though hashing and email have nothing to do with it

---

## Good Example — Fixed

Split each responsibility into its own type with a focused interface.

```go
package user

// --- Domain type ---

type User struct {
    Email    string
    Password string
}

// --- Repository: owns DB concern ---

type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) Save(u User) error {
    _, err := r.db.Exec("INSERT INTO users (email, password) VALUES (?, ?)", u.Email, u.Password)
    return err
}

// --- Hasher: owns security concern ---

type PasswordHasher struct{}

func (h *PasswordHasher) Hash(password string) string {
    sum := sha256.Sum256([]byte(password))
    return fmt.Sprintf("%x", sum)
}

// --- Mailer: owns email infrastructure concern ---

type WelcomeMailer struct {
    smtpAddr string
}

func (m *WelcomeMailer) Send(toEmail string) error {
    return smtp.SendMail(
        m.smtpAddr,
        nil,
        "no-reply@example.com",
        []string{toEmail},
        []byte("Welcome!"),
    )
}

// --- Coordinator: orchestrates, owns registration flow ---

type RegistrationService struct {
    repo   *UserRepository
    hasher *PasswordHasher
    mailer *WelcomeMailer
}

func (s *RegistrationService) Register(email, password string) error {
    u := User{
        Email:    email,
        Password: s.hasher.Hash(password),
    }
    if err := s.repo.Save(u); err != nil {
        return err
    }
    return s.mailer.Send(email)
}
```

**What changed:**
- Each type has exactly one reason to change
- `RegistrationService` coordinates them but owns only the registration flow
- You can test `PasswordHasher` with zero infrastructure
- Swapping SMTP for SendGrid only touches `WelcomeMailer`

---

## The "Reason to Change" Test

When you write a new type, ask:

> If I list all the people or teams who could ask me to change this code, how many are there?

- DB team asks to change the query → only `UserRepository` changes
- Security team asks to change hashing → only `PasswordHasher` changes
- Marketing changes the email copy → only `WelcomeMailer` changes

If every change is isolated to one type, SRP is satisfied.

---

## Go-Specific Notes

**Keep struct fields focused.** If a struct has fields from two unrelated domains
(e.g., `db *sql.DB` and `smtpClient *smtp.Client`), that's a signal it owns two concerns.

**Small files are natural in Go.** Go encourages many small files in one package.
Don't hesitate to split a 300-line file into three 100-line files by concern.

**Package-level SRP.** SRP applies to packages too, not just types. A package named
`util` that contains DB helpers, string formatters, and HTTP middleware violates SRP
at the package level.

---

## Common Mistake

Beginners confuse SRP with "a function should do one thing." That's good practice,
but SRP is at the **type/module level** — it's about cohesion of responsibilities,
not line count.

A `UserRepository` with `Save`, `FindByID`, `FindByEmail`, and `Delete` does NOT
violate SRP. All four methods serve the same responsibility: data persistence for users.
