// Package  sms delivers messages through the Africa's Talking SMS API and provides interfaces for SMS operations.
package sms

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Sender delivers a message to a phone number.
type Sender interface {
	Send(ctx context.Context, to, message string) error
}

// Client handles communication with the Africas Talking SMS API.
type Client struct {
	username string
	apiKey   string
	senderID string
	baseURL  string
	http     *http.Client
}

// New constructs an SMS Client targeting either production or sandbox endpoints
func New(username, apiKey, senderID string, sandbox bool) *Client {
	base := "https://api.africastalking.com"
	if sandbox {
		base = "https://api.sandbox.africastalking.com"
	}
	return &Client{
		username: username,
		apiKey:   apiKey,
		senderID: senderID,
		baseURL:  base,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Send posts an SMS message to a destination phone number.
func (c *Client) Send(ctx context.Context, to, message string) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	form := url.Values{}
	form.Set("username", c.username)
	form.Set("to", to)
	form.Set("message", message)
	if c.senderID != "" {
		form.Set("from", c.senderID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/version1/messaging", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apiKey", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sms: unexpected status %d", resp.StatusCode)
	}
	return nil
}
