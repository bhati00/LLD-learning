package main

import (
	"context"
	"fmt"
)

func main() {
	ns := NewNotificationService(3, 10)
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
	ns.Start()
	defer ns.Stop()

	n := Notification{
		userIDs:     []int{1, 2},
		messageType: Alert,
		priority:    High,
		channels:    []ChannelType{ChannelSMS, ChannelEmail, ChannelSlack},
		subject:     "Server down",
		body:        "Production API is not responding",
	}
	results := ns.Send(context.Background(), n)

	fmt.Println("\n--- results ---")
	for i := 0; i < len(n.userIDs)*len(n.channels); i++ {
		r := <-results
		status := "ok"
		if r.err != nil {
			status = "failed: " + r.err.Error()
		}
		fmt.Printf("user=%d channel=%s %s\n", r.userID, r.channel, status)
	}
}
