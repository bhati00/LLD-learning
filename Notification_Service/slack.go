package main

import (
	"context"
	"fmt"
)

type Slack struct {
	//  sms config
}

func (slack *Slack) send(ctx context.Context, recipient Recipient, sub string, body string) error {
	fmt.Print("i'm sendig slack msg to :", recipient.slackHandle)
	return nil
}
