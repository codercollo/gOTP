// Package sms provides implementations of the Sender interface for messagign providers and local testing.
package sms

import (
	"context"
	"io"
	"log"
)

// NoopSender provides a mock Sender implementation that logs outbound messages without dispatching them.
type NoopSender struct {
	Logger *log.Logger
}

// Send simulates sending a message by optinally logging the delivery parameters.
func (n NoopSender) Send(ctx context.Context, to, message string) error {
	if n.Logger != nil {
		n.Logger.Printf("[noop-sms] to=%s message=%q", to, message)
	}

	return nil
}

// Discard provides a pre-configured NoopSender that discards all log output.
var Discard Sender = NoopSender{
	Logger: log.New(io.Discard, "", 0),
}
