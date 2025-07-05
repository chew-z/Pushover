package main

import (
	"errors"
	"time"

	"github.com/gregdel/pushover"
)

// PushoverClient interface defines the methods we need from a Pushover client
type PushoverClient interface {
	SendMessage(message *pushover.Message, recipient *pushover.Recipient) (*pushover.Response, error)
}

// RealPushoverApp wraps a Pushover app to implement our PushoverClient interface
type RealPushoverApp struct {
	app *pushover.Pushover
}

// SendMessage implements the PushoverClient interface
func (r *RealPushoverApp) SendMessage(message *pushover.Message, recipient *pushover.Recipient) (*pushover.Response, error) {
	return r.app.SendMessage(message, recipient)
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

// CreateMessage creates a new Pushover message with the given parameters
func CreateMessage(text, title string, config Config, cliArgs *CLIArgs) *pushover.Message {
	// Use default title if none provided
	if title == "" && config.DefaultTitle != "" {
		title = config.DefaultTitle
	}

	message := pushover.NewMessageWithTitle(text, title)

	// Set priority (CLI args override config)
	priority := config.Priority
	if cliArgs.Priority != 0 {
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
	message.Expire = time.Duration(expireTime) * time.Second

	// Set device name (CLI args override config)
	deviceName := config.DeviceName
	if cliArgs.DeviceName != "" {
		deviceName = cliArgs.DeviceName
	}
	message.DeviceName = deviceName

	// Set timestamp to now
	message.Timestamp = time.Now().Unix()

	return message
}

// SendNotification sends a notification using the provided client
func SendNotification(client PushoverClient, message *pushover.Message, recipient *pushover.Recipient) error {
	_, err := client.SendMessage(message, recipient)
	return err
}
