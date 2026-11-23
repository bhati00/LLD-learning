package main

import (
	"context"
	"fmt"
)

type SMS struct {
	//  sms config
}

func (sms *SMS) send(ctx context.Context, recipient Recipient, sub string, body string) error {
	fmt.Print("i'm sendig msg to :", recipient.phone)
	return nil
}
