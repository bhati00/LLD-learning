package main

import (
	"fmt"
	"os"
	"sync"
)

// Destination is the extension point: implement it to log to Kafka, Slack, etc.
type Destination interface {
	Write(entry LogEntry) error
	Close() error
}

type ConsoleDestination struct{}

func NewConsole() *ConsoleDestination { return &ConsoleDestination{} }

func (c *ConsoleDestination) Write(entry LogEntry) error {
	_, err := fmt.Fprint(os.Stdout, entry.String())
	return err
}

func (c *ConsoleDestination) Close() error { return nil }

type FileDestination struct {
	mu sync.Mutex
	f  *os.File
}

func NewFile(path string) (*FileDestination, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileDestination{f: f}, nil
}

func (d *FileDestination) Write(entry LogEntry) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.f.WriteString(entry.String())
	return err
}

func (d *FileDestination) Close() error { return d.f.Close() }
