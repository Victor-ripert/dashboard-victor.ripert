// Package mailer sends emails. The rest of the gateway depends only on the
// Mailer interface, so the way emails are sent (SMTP provider, fake mailer in
// tests...) can be swapped without touching the features that send them.
package mailer

import "context"

type Message struct {
	To      string // "user@example.com" or "Name <user@example.com>"
	Subject string
	Body    string // plain text
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}
