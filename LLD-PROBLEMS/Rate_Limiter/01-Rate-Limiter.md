# Rate Limiter: Go Interview Roadmap

**For:** Go developers preparing for 3-4 YOE interviews.

Follow this order. Each topic's theory and implementation will be covered separately.

| Order | Topic | Material |
|---|---|---|
| 1 | Foundations: rate vs. concurrency, budget, policy, and scope | To cover separately |
| 2 | Token bucket | [Theory and diagrams](Token_Bucket/01-Token-Bucket.md), [implementation](Token_Bucket/tokenbucket.go), [demo](Token_Bucket/main.go) |
| 3 | Fixed-window counter | Next |
| 4 | Sliding-window log | Planned |
| 5 | Sliding-window counter | Planned |
| 6 | Per-user limiting: concurrent lookup, creation, and safe cleanup | Planned |
| 7 | HTTP middleware and `golang.org/x/time/rate` | Planned |
| 8 | Distributed limiting: replicas, Redis atomicity, and failure policy | Planned |
| 9 | Leaky bucket and traffic shaping | Planned |

For each implementation, cover configuration validation, Go concurrency safety, and focused tests before moving on.
