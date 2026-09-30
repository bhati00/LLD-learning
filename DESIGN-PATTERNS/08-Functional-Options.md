# Functional Options Pattern

> **Configure a Go value with optional functions passed to its constructor.**

Functional options keep constructors readable as configuration grows while preserving sensible defaults. They are common in public Go APIs where callers need different subsets of optional settings.

---

## The Core Idea

The constructor creates a valid default value, then applies each option. An option is a function that changes one part of the configuration.

---

## Example

```go
package client

import (
	"errors"
	"time"
)

type Client struct {
	endpoint string
	timeout  time.Duration
	retries  int
}

type Option func(*Client) error

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		if timeout <= 0 {
			return errors.New("timeout must be positive")
		}
		c.timeout = timeout
		return nil
	}
}

func WithRetries(retries int) Option {
	return func(c *Client) error {
		if retries < 0 {
			return errors.New("retries cannot be negative")
		}
		c.retries = retries
		return nil
	}
}

func New(endpoint string, options ...Option) (*Client, error) {
	c := &Client{
		endpoint: endpoint,
		timeout:  5 * time.Second,
		retries:  2,
	}
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}
```

Usage remains self-documenting:

```go
c, err := client.New("https://api.example.com",
	client.WithTimeout(2*time.Second),
	client.WithRetries(4),
)
```

---

## When It Helps

- A constructor has multiple optional settings with good defaults.
- Callers commonly specify different subsets of options.
- Options benefit from descriptive names and validation.

## Costs and Pitfalls

- For one or two simple fields, a config struct may be more direct.
- Hidden mutation and option ordering can surprise callers; document interactions.
- Decide whether options can fail. If invalid configuration is possible, return errors rather than silently ignoring it.
- Do not expose options for every internal field; that makes the API difficult to maintain.
- Avoid options that mutate global state or perform surprising I/O during construction.

## Alternative: Config Struct

Use a config struct when callers routinely set many fields together, need to serialize configuration, or benefit from seeing the complete configuration as data. Functional options work best for ergonomic, mostly optional constructor settings.

## Interview Check

Explain defaults, validation, and option ordering. Compare functional options to a config struct and justify the public API based on the number and shape of optional settings.