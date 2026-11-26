package main

import (
	"fmt"
	"os"
	"time"
)

type Level int

const (
	Debug Level = iota + 1
	Info
	Warn
	Error
	Fatal
)

func (lv Level) String() string {
	switch lv {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	case Fatal:
		return "FATAL"
	}
	return "UNKNOWN"
}

// LogEntry is exported so destinations and formatters can live in other packages.
type LogEntry struct {
	Timestamp time.Time
	Level     Level
	Message   string
}

func (e LogEntry) String() string {
	return fmt.Sprintf("%s [%s] %s\n", e.Timestamp.Format(time.DateTime), e.Level, e.Message)
}

// Logger decides whether to log; each Destination decides where it goes.
type Logger struct {
	dests    []Destination
	minLevel Level
}

type Option func(*Logger)

func WithMinLevel(lv Level) Option {
	return func(l *Logger) { l.minLevel = lv }
}

func New(dests []Destination, opts ...Option) *Logger {
	l := &Logger{dests: dests, minLevel: Debug}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// A failing destination must not stop the others, so errors are reported and the loop continues.
func (l *Logger) log(lv Level, message string) {
	if lv < l.minLevel {
		return
	}
	entry := LogEntry{Timestamp: time.Now(), Level: lv, Message: message}
	for _, d := range l.dests {
		if err := d.Write(entry); err != nil {
			fmt.Fprintf(os.Stderr, "logger: write failed: %v\n", err)
		}
	}
}

func (l *Logger) Close() {
	for _, d := range l.dests {
		d.Close()
	}
}

func (l *Logger) Debug(message string) { l.log(Debug, message) }
func (l *Logger) Info(message string)  { l.log(Info, message) }
func (l *Logger) Warn(message string)  { l.log(Warn, message) }
func (l *Logger) Error(message string) { l.log(Error, message) }
func (l *Logger) Fatal(message string) { l.log(Fatal, message) }
