package main

import (
	"context"
	"fmt"
)

type NotificationService struct {
	channels map[ChannelType]Channel
	users    map[int]Recipient
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		channels: make(map[ChannelType]Channel),
		users:    make(map[int]Recipient),
	}
}

func (ns *NotificationService) RegisterChannel(t ChannelType, ch Channel) {
	ns.channels[t] = ch
}

func (ns *NotificationService) AddUser(r Recipient) {
	ns.users[r.id] = r
}

// Send delivers n to every user on every requested channel and returns one
// Result per (user, channel) pair. Failures are reported in the results,
// they never stop the remaining sends.
func (ns *NotificationService) Send(ctx context.Context, n Notification) []Result {
	results := make([]Result, 0, len(n.userIDs)*len(n.channels))

	for _, id := range n.userIDs {
		recipient, userFound := ns.users[id]

		for _, t := range n.channels {
			var err error
			switch ch, channelFound := ns.channels[t]; {
			case !userFound:
				err = fmt.Errorf("user %d not found", id)
			case !channelFound:
				err = fmt.Errorf("no channel registered for %s", t)
			default:
				err = ch.send(ctx, recipient, n.subject, n.body)
			}
			results = append(results, Result{userID: id, channel: t, err: err})
		}
	}
	return results
}
