# Logger: Approach, Learnings and Interview Follow-ups

[Back to LLD topics](../LLD-Topics.md)

**Target:** 3-4 YOE Go developer. Goal: design a logger on a whiteboard, explain why it is shaped this way, and survive the follow-ups.

Code lives in this folder: `logger.go`, `destination.go`, `main.go`.

---

## 1. The Problem

Application code says "log this message". Where it goes (console, file, Kafka, Slack) and how it looks (text, JSON) should not be the application's concern.

The design question is: **how do I add new destinations without editing the logger?**

Analogy: the logger is a waiter. The customer says "deliver this". The waiter should not be hard-wired to one table. The waiter should accept any table that follows a standard shape.

---

## 2. The Key Insight: Two Separate Jobs

We first asked "interface or functional options?". The answer is both, because they solve different problems.

| Job | Question | Tool |
|---|---|---|
| Extensibility | What *is* a destination? | **Interface** (`Destination`) |
| Configuration | How do I set the logger up? | **Options** (`WithMinLevel`) |

- **Interface** is the socket shape. Anyone can build a plug (Kafka, Slack) and the logger never changes. This is Open/Closed and the Strategy pattern.
- **Options** are for knobs: minimum level, and so on. They cannot make the logger extensible on their own. If you only use options, you end up with `WithFile`, `WithKafka`, `WithSlack` inside your package, a closed list that every new destination has to edit.

Our first code had `WithFile` hard-wired into the logger package. That is the extensibility problem.

---

## 3. The Final Design

```go
type Destination interface {
    Write(entry LogEntry) error
    Close() error
}

type Logger struct {
    dests    []Destination
    minLevel Level
}

func New(dests []Destination, opts ...Option) *Logger
func WithMinLevel(lv Level) Option
```

Usage:

```go
file, _ := NewFile("app.log")
log := New([]Destination{NewConsole(), file}, WithMinLevel(Info))
defer log.Close()
```

Flow of one log call:

1. `log.Info("hi")` runs.
2. The logger checks its own setting. If the level is below `minLevel`, it stops.
3. It builds one `LogEntry` (time, level, message).
4. It loops over the destinations and calls `Write`.
5. If one destination fails, the error goes to stderr and the loop **continues**. One bad destination must not stop the others.

Rule for where a setting goes: if it applies to the whole logger, it is a logger option. If it applies to one destination, it belongs to that destination.

---

## 4. What Real Libraries Do

We checked how the popular Go loggers handle this.

| Library | Adding destinations | Extension point |
|---|---|---|
| zerolog | `zerolog.New(zerolog.MultiLevelWriter(a, b))` | `io.Writer` |
| zap | `zapcore.NewTee(core1, core2)` | `zapcore.Core` interface |
| slog (standard library) | `slog.New(handler)` | `slog.Handler` interface |
| logrus | `SetOutput(w)` and `AddHook(h)` | `io.Writer` and `Hook` |

Patterns:

- The **interface is always the extension point**.
- **Fan-out is separate and composable** (`MultiLevelWriter`, `NewTee`). The logger itself only talks to one thing.
- **Options configure behaviour**, not destinations. The destination (core, writer, handler) is a required argument.

---

## 5. Decisions We Made and Why

### Required destination, not an option

A logger with no destination is meaningless. `New(WithDestination(a), WithDestination(b))` allows zero destinations and a silent default. Passing it as an argument removes that state.

### Slice in the Logger vs a `NewMulti` wrapper

We built `NewMulti(dests...)` first. It is itself a `Destination`, so it nests and can be wrapped (`NewAsync`, `NewFiltered`). The logger then talks to one destination and stays simple. That is the production-grade design and matches zap's `Tee`.

For the **interview version** we use a plain `[]Destination` in the Logger. It is faster to write, easy to explain, and enough for the core. `NewMulti` is something to *say*, not to code.

### Formatter interface: cut for the interview version

We built `Formatter`, `TextFormatter`, `JSONFormatter` and a per-destination `WithFormatter` option. The idea is that each destination chooses its own format (readable text for console, JSON for a file), so format is a destination setting, not a logger setting. `Destination.Write` takes the raw `LogEntry`, not a string, so each destination can format independently.

We removed it for the interview version. Format now lives in `LogEntry.String()`. Mention the `Formatter` as an extension.

### `New` returning no error

Opening a file can fail, so `NewFile(path)` returns the error. That keeps `New` infallible and simpler for users.

### Exported `LogEntry` fields

Destinations written by other people, in other packages, need to read the entry. Lower-case fields would block them.

### Mutex on the destination, not the logger

A single logger-wide lock would let a slow destination (network) block all the others. Each destination guards itself.

- `os.File.Write` is already safe for a single call, so the console needs no mutex. We removed it.
- `FileDestination` keeps its mutex as a defensive measure. The `Destination` interface is public, so each implementation should own its thread safety. The lock becomes necessary once you wrap an arbitrary `io.Writer` (a `bytes.Buffer` is not goroutine-safe) or do two writes per entry.
- The lock is per instance. Two separate destinations do not protect each other.

---

## 6. Interview Cheat Sheet

**What to write live (about 15 minutes)**

- `Level` and the log methods.
- `Destination` interface with `Write` and `Close`.
- `ConsoleDestination` and `FileDestination`.
- `Logger` holding `[]Destination`, `minLevel` check, loop with error handling.

**What to say out loud**

- A `Formatter` interface for per-destination text or JSON.
- `NewMulti` and other decorators, since a wrapper is also a `Destination`.
- `Close()` for files and connections.
- Slow destinations need their own buffer or goroutine.

---

## 7. Follow-up Questions

**1. What if a destination is slow (Kafka, Slack)?**
The caller currently waits for every write. Wrap the destination so `Write` pushes into a buffered channel and a background goroutine does the real write. Decide what happens when the buffer is full (drop or block), and flush on `Close`. Use a single worker to keep ordering. This is the most common follow-up.

**2. How do I send only errors to Slack but everything to the file?**
The min level is logger-wide now. Put a level on the destination, or wrap it in a `NewFiltered(dest, Warn)` decorator.

**3. How do I support JSON versus text?**
Add a `Formatter` interface and give each destination its own. `Write` takes the raw entry so each destination formats independently.

**4. Does `Fatal` exit?**
Ours does not. Real loggers call `os.Exit(1)` after writing, and `Panic` panics. It is a one-line change.

**5. Can this be a global logger?**
A package-level default created with `sync.Once`, while the injectable logger stays available for tests.

**6. How would you test it?**
`ConsoleDestination` writes to `os.Stdout`, which is hard to assert on. Accept an `io.Writer`, or test with a fake `Destination` that records entries.

**7. Structured fields (`user_id=42`) or caller file and line?**
Add `Fields map[string]any` to `LogEntry`, and `runtime.Caller` for the source location.

**8. Can the level change at runtime?**
Store it as an `atomic.Int32` instead of a plain field.

**9. What about file rotation?**
Rotate by size or date, or use a library such as lumberjack, behind the same `Destination` interface.

**10. Is line ordering guaranteed?**
Within one destination, yes: the lock serialises writes. Across destinations, no. An async version needs a single worker to keep order.

**11. Why is a mutex in the file destination but not the console?**
`os.File.Write` is safe for a single call, so it is redundant in both. We kept it for the file as a defensive habit and because the interface is public. See section 5.

**12. What happens if one destination fails?**
The logger reports the error to stderr and continues with the remaining destinations. With `NewMulti` the errors are joined with `errors.Join`.

---

## 8. Key Learnings

1. Separate **what varies** (destination, format) from **what is fixed** (the logger's job). Put an interface on what varies.
2. Interfaces give **extensibility**. Options give **clean configuration**. Do not ask one to do the other's job.
3. A setting belongs where it applies: logger-wide in the logger, per-destination in the destination.
4. Required things are arguments. Optional things are options.
5. Fan-out and wrappers are best done as a `Destination` that wraps other destinations (decorator).
6. For interviews, write the core and **talk through** the extensions. Time is the constraint.
7. Do not add a lock because it feels safe. Know what is already safe (`os.File.Write`) and why you add a lock anyway.
8. Check how real libraries (zap, zerolog, slog) solve the same problem before inventing your own API.
