package main

import (
	"context"
	"fmt"
)

type Slack struct {
	//  sms config
}

func (slack *Slack) send(ctx context.Context, recipient Recipient, sub string, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fmt.Print("i'm sendig slack msg to :", recipient.slackHandle)
	return nil
}
