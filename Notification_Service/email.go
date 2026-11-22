package main

import (
	"context"
	"fmt"
)

type Email struct {
	//  sms config
}

func (email *Email) send(ctx context.Context, recipient Recipient, sub string, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fmt.Print("i'm sendig email to :", recipient.email)
	return nil
}
