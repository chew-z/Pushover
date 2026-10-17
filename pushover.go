package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gregdel/pushover"
)

// maxExpireSeconds is Pushover's maximum emergency notification retry duration.
const maxExpireSeconds = 10800

// PushoverClient interface defines the methods we need from a Pushover client
type PushoverClient interface {
	SendMessage(message *pushover.Message, recipient *pushover.Recipient) (*pushover.Response, error)
	CancelEmergencyNotification(receipt string) (*pushover.Response, error)
	SendGlanceUpdate(glance *pushover.Glance, recipient *pushover.Recipient) (*pushover.Response, error)
}

// RealPushoverApp wraps a Pushover app to implement our PushoverClient interface
type RealPushoverApp struct {
	app *pushover.Pushover
}

// SendMessage implements the PushoverClient interface
func (r *RealPushoverApp) SendMessage(message *pushover.Message, recipient *pushover.Recipient) (*pushover.Response, error) {
	return r.app.SendMessage(message, recipient)
}

// CancelEmergencyNotification cancels the emergency notification identified by receipt.
func (r *RealPushoverApp) CancelEmergencyNotification(receipt string) (*pushover.Response, error) {
	return r.app.CancelEmergencyNotification(receipt)
}

// SendGlanceUpdate sends a Glances update to the recipient.
func (r *RealPushoverApp) SendGlanceUpdate(glance *pushover.Glance, recipient *pushover.Recipient) (*pushover.Response, error) {
	return r.app.SendGlanceUpdate(glance, recipient)
}

// NewPushoverClient creates a new Pushover client with the given configuration
func NewPushoverClient(config Config) (PushoverClient, *pushover.Recipient, error) {
	appKey := config.AppKey
	recipientKey := config.RecipientKey

	// Validate required keys
	if appKey == "" {
		return nil, nil, errors.New("APP_KEY environment variable is required")
	}

	if recipientKey == "" {
		return nil, nil, errors.New("RECIPIENT_KEY environment variable is required")
	}

	// Create the app and recipient
	app := pushover.New(appKey)
	recipient := pushover.NewRecipient(recipientKey)

	return &RealPushoverApp{app: app}, recipient, nil
}

// newPushoverClient creates a Pushover client and lets tests substitute a mock.
var newPushoverClient = NewPushoverClient

// CreateMessage creates a Pushover message and returns an error for invalid input.
func CreateMessage(text, title string, config Config, cliArgs *CLIArgs) (*pushover.Message, error) {
	if cliArgs.HTML && cliArgs.Monospace {
		return nil, errors.New("html and monospace are mutually exclusive")
	}
	if cliArgs.TTL < 0 {
		return nil, errors.New("ttl must be at least 1 second")
	}
	if cliArgs.Timestamp < 0 {
		return nil, errors.New("timestamp must be a positive Unix time in seconds")
	}
	if utf8.RuneCountInString(cliArgs.URL) > 512 {
		return nil, errors.New("url must be at most 512 characters")
	}
	if utf8.RuneCountInString(cliArgs.URLTitle) > 100 {
		return nil, errors.New("url_title must be at most 100 characters")
	}
	if cliArgs.URLTitle != "" && cliArgs.URL == "" {
		return nil, errors.New("url_title requires url")
	}

	// Use default title if none provided
	if title == "" && config.DefaultTitle != "" {
		title = config.DefaultTitle
	}

	message := pushover.NewMessageWithTitle(text, title)

	// Set priority when the caller explicitly supplied one, including zero.
	priority := config.Priority
	if cliArgs.PrioritySet {
		priority = cliArgs.Priority
	}
	message.Priority = priority

	// Set sound (CLI args override config)
	sound := config.Sound
	if cliArgs.Sound != "" {
		sound = cliArgs.Sound
	}
	message.Sound = sound

	// Set expire time (CLI args override config)
	expireTime := config.ExpireTime
	if cliArgs.ExpireTime != 0 {
		expireTime = cliArgs.ExpireTime
	}
	if expireTime > maxExpireSeconds {
		return nil, fmt.Errorf("expire must be at most %d seconds", maxExpireSeconds)
	}
	message.Expire = time.Duration(expireTime) * time.Second

	// Set retry interval (CLI args override config; required for emergency priority)
	retryTime := config.RetryTime
	if cliArgs.RetryTime != 0 {
		retryTime = cliArgs.RetryTime
	}
	message.Retry = time.Duration(retryTime) * time.Second

	// Set device name (CLI args override config)
	deviceName := config.DeviceName
	if cliArgs.DeviceName != "" {
		deviceName = cliArgs.DeviceName
	}
	message.DeviceName = deviceName
	message.URL = cliArgs.URL
	message.URLTitle = cliArgs.URLTitle
	message.HTML = cliArgs.HTML
	message.Monospace = cliArgs.Monospace
	if cliArgs.TTL > 0 {
		message.TTL = time.Duration(cliArgs.TTL) * time.Second
	}
	if cliArgs.Timestamp > 0 {
		message.Timestamp = cliArgs.Timestamp
	}

	return message, nil
}

// SendNotification sends a notification using the provided client
func SendNotification(client PushoverClient, message *pushover.Message, recipient *pushover.Recipient) error {
	_, err := client.SendMessage(message, recipient)
	return err
}

// formatSendResult formats a successful Pushover response for a caller.
func formatSendResult(resp *pushover.Response, actor string) string {
	parts := []string{"Notification sent"}
	if resp != nil && resp.Receipt != "" {
		parts = append(parts, "Receipt: "+resp.Receipt)
	}
	if resp != nil && resp.Limit != nil {
		quota := "Quota: " + strconv.Itoa(resp.Limit.Remaining) + "/" + strconv.Itoa(resp.Limit.Total) + " remaining"
		if !resp.Limit.NextReset.IsZero() {
			quota += ", resets " + resp.Limit.NextReset.UTC().Format(time.RFC3339)
		}
		parts = append(parts, quota)
	}

	text := strings.Join(parts, ". ")
	if len(parts) == 1 {
		text += "."
	}
	if actor != "" {
		text += " (sent by " + actor + ")"
	}

	return text
}
