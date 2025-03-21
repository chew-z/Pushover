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

// LoadConfig loads the configuration from environment variables with fallbacks
func LoadConfig() Config {
	return Config{
		AppKey:       getEnvWithFallback("APP_KEY", ""),
		RecipientKey: getEnvWithFallback("RECIPIENT_KEY", ""),
		DeviceName:   os.Getenv("DEVICE_NAME"),
		DefaultTitle: "",
	}
}

// getEnvWithFallback returns the environment variable value or the fallback if not set
func getEnvWithFallback(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
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
func NewPushoverClient(config Config) (PushoverClient, *pushover.Recipient, error) {
	appKey := config.AppKey
	recipientKey := config.RecipientKey
	
	// Apply fallbacks for required keys
	if appKey == "" {
		appKey = "a84t4wvdijbn3pcjtpbhnb1vivdn7m" // Fallback app key
	}
	
	if recipientKey == "" {
		recipientKey = "gqq7a1wbs4b7nkahg6nemr7jyfprox" // Fallback recipient key
	}
	
	// Create the app and recipient
	app := pushover.New(appKey)
	recipient := pushover.NewRecipient(recipientKey)
	
	return &RealPushoverApp{app: app}, recipient, nil
}

// CreateMessage creates a new Pushover message with the given parameters
func CreateMessage(text, title string, config Config) *pushover.Message {
	message := pushover.NewMessageWithTitle(text, title)
	message.Priority = pushover.PriorityLow
	message.Timestamp = time.Now().Unix()
	message.Expire = 180
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
	client, recipient, err := NewPushoverClient(config)
	if err != nil {
		log.Printf("failed to create Pushover client: %v", err)
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
