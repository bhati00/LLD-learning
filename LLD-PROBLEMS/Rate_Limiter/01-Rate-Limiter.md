# Rate Limiter: Go Interview Roadmap

**For:** Go developers preparing for 3-4 YOE interviews.

Follow this order. Each topic's theory and implementation will be covered separately.

| Order | Topic | Material |
|---|---|---|
| 1 | Foundations: rate vs. concurrency, budget, policy, and scope | To cover separately |
| 2 | Token bucket | [Theory and diagrams](Token_Bucket/01-Token-Bucket.md), [implementation](Token_Bucket/tokenbucket.go), [demo](Token_Bucket/main.go) |
| 3 | Fixed-window counter | [Theory and flow diagram](Fixed_Window_Counter/01-Fixed-Window-Counter.md), [implementation](Fixed_Window_Counter/fixedWindow.go), [tests](Fixed_Window_Counter/fixedWindow_test.go) |
| 4 | Sliding-window log | [Theory and flow diagram](Sliding_Window_Log/01-Sliding-Window-Log.md), [implementation](Sliding_Window_Log/Sliding_window.go), [tests](Sliding_Window_Log/Sliding_window_test.go) |
| 5 | Sliding-window counter | Planned |
| 6 | Per-user HTTP rate limiter: concurrent registry, middleware, `golang.org/x/time/rate` comparison, tests, and safe cleanup | [Theory and flow diagram](Per_User_HTTP_Limiter/01-Per-User-HTTP-Limiter.md); implementation in stages |
| 7 | Distributed limiting: replicas, Redis atomicity, and failure policy | Planned |
| 8 | Leaky bucket and traffic shaping | Planned |

For each implementation, cover configuration validation, Go concurrency safety, and focused tests before moving on.
