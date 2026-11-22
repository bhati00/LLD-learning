package main

type Priority int

const (
	Low Priority = iota
	Medium
	High
)

type MessageType int

const (
	Marketing MessageType = 1 + iota
	OTP
	Alert
	Transactional
)

// Prefixed so they don't clash with the SMS/Email/Slack channel structs later.
type ChannelType int

const (
	ChannelSMS ChannelType = iota + 1
	ChannelEmail
	ChannelSlack
)

func (c ChannelType) String() string {
	switch c {
	case ChannelSMS:
		return "sms"
	case ChannelEmail:
		return "email"
	case ChannelSlack:
		return "slack"
	}
	return "unknown"
}

// Recipient holds contact details; each channel picks the field it needs.
type Recipient struct {
	id          int
	name        string
	email       string
	phone       string
	slackHandle string
}

// Notification is the caller's request. It carries user IDs, not Recipients;
// the service resolves them through a user store.
type Notification struct {
	userIDs     []int
	messageType MessageType
	priority    Priority
	channels    []ChannelType
	subject     string // only used by channels that have one (email)
	body        string
}

// Result is the outcome of sending to one user on one channel.
type Result struct {
	userID  int
	channel ChannelType
	err     error
}
