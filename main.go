package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
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
	Priority     int
	Sound        string
	ExpireTime   int
}

// CLIArgs holds parsed command-line arguments
type CLIArgs struct {
	Message     string
	Title       string
	Priority    int
	Sound       string
	ExpireTime  int
	DeviceName  string
	ShowVersion bool
	ShowHelp    bool
}

const version = "1.0.0"

// LoadConfig loads the configuration from environment variables
func LoadConfig() Config {
	// Parse priority with error handling, default to PriorityLow
	priority := int(pushover.PriorityLow)
	if p, err := strconv.Atoi(os.Getenv("PUSHOVER_PRIORITY")); err == nil && p != 0 {
		priority = p
	}

	sound := os.Getenv("PUSHOVER_SOUND")
	if sound == "" {
		sound = pushover.SoundVibrate
	}

	// Parse expire time with error handling, default to 180 seconds
	expireTime := 180
	if e, err := strconv.Atoi(os.Getenv("PUSHOVER_EXPIRE")); err == nil && e != 0 {
		expireTime = e
	}

	return Config{
		AppKey:       os.Getenv("APP_KEY"),
		RecipientKey: os.Getenv("RECIPIENT_KEY"),
		DeviceName:   os.Getenv("DEVICE_NAME"),
		DefaultTitle: os.Getenv("DEFAULT_TITLE"),
		Priority:     priority,
		Sound:        sound,
		ExpireTime:   expireTime,
	}
}

// ParseArgs parses command-line arguments using the flag package
func ParseArgs(args []string) (*CLIArgs, error) {
	fs := flag.NewFlagSet("pushover", flag.ContinueOnError)

	cliArgs := &CLIArgs{}

	fs.StringVar(&cliArgs.Message, "m", "", "Message to send (required)")
	fs.StringVar(&cliArgs.Message, "message", "", "Message to send (required)")
	fs.StringVar(&cliArgs.Title, "t", "", "Message title")
	fs.StringVar(&cliArgs.Title, "title", "", "Message title")
	fs.IntVar(&cliArgs.Priority, "p", 0, "Priority (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)")
	fs.IntVar(&cliArgs.Priority, "priority", 0, "Priority (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)")
	fs.StringVar(&cliArgs.Sound, "s", "", "Sound name")
	fs.StringVar(&cliArgs.Sound, "sound", "", "Sound name")
	fs.IntVar(&cliArgs.ExpireTime, "e", 0, "Expire time in seconds")
	fs.IntVar(&cliArgs.ExpireTime, "expire", 0, "Expire time in seconds")
	fs.StringVar(&cliArgs.DeviceName, "d", "", "Device name")
	fs.StringVar(&cliArgs.DeviceName, "device", "", "Device name")
	fs.BoolVar(&cliArgs.ShowVersion, "version", false, "Show version")
	fs.BoolVar(&cliArgs.ShowHelp, "h", false, "Show help")
	fs.BoolVar(&cliArgs.ShowHelp, "help", false, "Show help")

	// Custom usage function
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [OPTIONS]\n", args[0])
		fmt.Fprintln(fs.Output(), "\nSend push notifications via Pushover")
		fmt.Fprintln(fs.Output(), "\nOptions:")
		fs.PrintDefaults()
		fmt.Fprintln(fs.Output(), "\nEnvironment Variables:")
		fmt.Fprintln(fs.Output(), "  APP_KEY           Pushover application key (required)")
		fmt.Fprintln(fs.Output(), "  RECIPIENT_KEY     Pushover recipient key (required)")
		fmt.Fprintln(fs.Output(), "  DEVICE_NAME       Default device name")
		fmt.Fprintln(fs.Output(), "  DEFAULT_TITLE     Default message title")
		fmt.Fprintln(fs.Output(), "  PUSHOVER_PRIORITY Default priority")
		fmt.Fprintln(fs.Output(), "  PUSHOVER_SOUND    Default sound")
		fmt.Fprintln(fs.Output(), "  PUSHOVER_EXPIRE   Default expire time")
	}

	if err := fs.Parse(args[1:]); err != nil {
		return nil, err
	}

	if cliArgs.ShowHelp {
		fs.Usage()
		return cliArgs, nil
	}

	if cliArgs.ShowVersion {
		fmt.Printf("pushover version %s\n", version)
		return cliArgs, nil
	}

	// Check for positional arguments as fallback
	remaining := fs.Args()
	if cliArgs.Message == "" && len(remaining) > 0 {
		cliArgs.Message = remaining[0]
		if cliArgs.Title == "" && len(remaining) > 1 {
			cliArgs.Title = remaining[1]
		}
	}

	if cliArgs.Message == "" {
		return nil, errors.New("message is required")
	}

	return cliArgs, nil
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

// Run encapsulates the main application logic for easier testing
func Run(args []string, client PushoverClient) error {
	// Load configuration
	config := LoadConfig()

	// Parse command-line arguments
	cliArgs, err := ParseArgs(args)
	if err != nil {
		return fmt.Errorf("argument parsing failed: %w", err)
	}

	// Handle help and version flags
	if cliArgs.ShowHelp || cliArgs.ShowVersion {
		return nil // Already handled in ParseArgs
	}

	// Create Pushover client if not provided (for testing)
	var recipient *pushover.Recipient
	if client == nil {
		client, recipient, err = NewPushoverClient(config)
		if err != nil {
			return fmt.Errorf("failed to create Pushover client: %w", err)
		}
	} else {
		// For testing, create a dummy recipient
		recipient = pushover.NewRecipient("dummy")
	}

	// Create the message
	pushMessage := CreateMessage(cliArgs.Message, cliArgs.Title, config, cliArgs)

	// Send the notification
	if err := SendNotification(client, pushMessage, recipient); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Println("Notification sent successfully")
	return nil
}

func main() {
	// Configure logging
	log.SetFlags(0) // Clean output without timestamps

	err := Run(os.Args, nil)
	if err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
