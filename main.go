package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gregdel/pushover"
	_ "github.com/joho/godotenv/autoload"
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

// Config holds the application configuration
type Config struct {
	AppKey       string
	RecipientKey string
	DeviceName   string
	DefaultTitle string
}

// LoadConfig loads the configuration from environment variables
func LoadConfig() Config {
	return Config{
		AppKey:       os.Getenv("APP_KEY"),
		RecipientKey: os.Getenv("RECIPIENT_KEY"),
		DeviceName:   os.Getenv("DEVICE_NAME"),
		DefaultTitle: "",
	}
}

// ParseArgs parses command-line arguments and returns the message and title
func ParseArgs(args []string) (message, title string, err error) {
	if len(args) < 2 {
		return "", "", errors.New("usage: push <message> [<title>]")
	}

	message = args[1]
	if len(args) > 2 {
		title = args[2]
	}

	return message, title, nil
}

// NewPushoverClient creates a new Pushover client with the given configuration
func NewPushoverClient(config Config) (PushoverClient, *pushover.Recipient) {
	appKey := config.AppKey
	recipientKey := config.RecipientKey

	// Validate required keys
	if appKey == "" {
		log.Println("Error: APP_KEY environment variable is required")
		return nil, nil
	}

	if recipientKey == "" {
		log.Println("Error: RECIPIENT_KEY environment variable is required")
		return nil, nil
	}

	// Create the app and recipient
	app := pushover.New(appKey)
	recipient := pushover.NewRecipient(recipientKey)

	return &RealPushoverApp{app: app}, recipient
}

// CreateMessage creates a new Pushover message with the given parameters
func CreateMessage(text, title string, config Config) *pushover.Message {
	message := pushover.NewMessageWithTitle(text, title)
	message.Priority = pushover.PriorityLow
	message.Timestamp = time.Now().Unix()
	message.Expire = time.Duration(180 * time.Second)
	message.DeviceName = config.DeviceName
	message.Sound = pushover.SoundVibrate

	return message
}

// SendNotification sends a notification using the provided client
func SendNotification(client PushoverClient, message *pushover.Message, recipient *pushover.Recipient) error {
	_, err := client.SendMessage(message, recipient)
	return err
}

func main() {
	// Load configuration
	config := LoadConfig()

	// Parse command-line arguments
	message, title, err := ParseArgs(os.Args)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Create Pushover client
	client, recipient := NewPushoverClient(config)
	if client == nil || recipient == nil {
		os.Exit(1)
	}

	// Create the message
	pushMessage := CreateMessage(message, title, config)

	// Send the notification
	if err := SendNotification(client, pushMessage, recipient); err != nil {
		log.Printf("failed to send message: %v", err)
		os.Exit(1)
	}

	fmt.Println("Notification sent successfully")
}
