package main

import (
	"context"
	"fmt"
)

func main() {
	ns := NewNotificationService()
	ns.AddUser(Recipient{
		id:          1,
		name:        "lalu",
		email:       "lalu@gmail.com",
		phone:       "123455",
		slackHandle: "@lalu",
	})
	ns.AddUser(Recipient{
		id:          2,
		name:        "kalu",
		email:       "kalu@gmail.com",
		phone:       "987654",
		slackHandle: "@kalu",
	})
	ns.RegisterChannel(ChannelSMS, &SMS{})
	ns.RegisterChannel(ChannelSlack, &Slack{})
	ns.RegisterChannel(ChannelEmail, &Email{})
	n := Notification{
		userIDs:     []int{1, 2, 3}, // 3 does not exist, so it should fail
		messageType: Alert,
		priority:    High,
		channels:    []ChannelType{ChannelSMS, ChannelEmail, ChannelSlack},
		subject:     "Server down",
		body:        "Production API is not responding",
	}
	results := ns.Send(context.Background(), n)

	fmt.Println("\n--- results ---")
	for _, r := range results {
		status := "ok"
		if r.err != nil {
			status = "failed: " + r.err.Error()
		}
		fmt.Printf("user=%d channel=%s %s\n", r.userID, r.channel, status)
	}
}
