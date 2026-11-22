package main

import "context"

type Channel interface {
	send(ctx context.Context, recipient Recipient, sub string, body string) error
}
