package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/gregdel/pushover"
)

// CLIArgs holds parsed command-line arguments
type CLIArgs struct {
	Message  string
	Title    string
	Priority int
	// PrioritySet is true when the caller explicitly supplied a priority.
	PrioritySet bool
	// URL is the supplementary URL shown with the notification.
	URL string
	// URLTitle is the title shown for URL.
	URLTitle string
	// HTML renders the message as HTML.
	HTML bool
	// Monospace renders the message in a monospace font.
	Monospace bool
	// TTL is seconds until the notification is deleted; 0 means unset.
	TTL int
	// Timestamp is Unix seconds to display; 0 means unset.
	Timestamp int64
	// AttachmentPath is the local path of the image attachment.
	AttachmentPath string
	// AttachmentType is the expected MIME type of the image attachment.
	AttachmentType string
	// Attachment contains the loaded image attachment bytes.
	Attachment  []byte
	Sound       string
	ExpireTime  int
	RetryTime   int
	DeviceName  string
	ShowVersion bool
	ShowHelp    bool
}

// showMainHelp writes the main help text, including subcommands, to the FlagSet's output.
func showMainHelp(fs *flag.FlagSet, progName string) {
	out := fs.Output()
	fmt.Fprintf(out, "Usage: %s [OPTIONS] | %s SUBCOMMAND [OPTIONS]\n", progName, progName)
	fmt.Fprintln(out, "\nSend push notifications via Pushover")
	fmt.Fprintln(out, "\nSubcommands:")
	fmt.Fprintln(out, "  cancel            Cancel an emergency notification by receipt")
	fmt.Fprintln(out, "  glance            Send an Apple Watch complication Glance update")
	fmt.Fprintln(out, "  mcp               Start MCP server mode")
	fmt.Fprintln(out, "\nOptions:")
	fs.PrintDefaults()
	fmt.Fprintln(out, "\nEnvironment Variables:")
	fmt.Fprintln(out, "  APP_KEY           Pushover application key (required)")
	fmt.Fprintln(out, "  RECIPIENT_KEY     Pushover recipient key (required)")
	fmt.Fprintln(out, "  DEVICE_NAME       Default device name")
	fmt.Fprintln(out, "  DEFAULT_TITLE     Default message title")
	fmt.Fprintln(out, "  PUSHOVER_PRIORITY Default priority")
	fmt.Fprintln(out, "  PUSHOVER_SOUND    Default sound")
	fmt.Fprintln(out, "  PUSHOVER_EXPIRE   Default expire time")
	fmt.Fprintln(out, "  PUSHOVER_RETRY    Default retry interval for emergency messages")
}

// ParseArgs parses command-line arguments using the flag package
func ParseArgs(args []string) (*CLIArgs, error) {
	if len(args) < 1 {
		return nil, errors.New("no program name in arguments")
	}
	fs := flag.NewFlagSet("pushover", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cliArgs := &CLIArgs{}

	fs.StringVar(&cliArgs.Message, "m", "", "Message to send (required)")
	fs.StringVar(&cliArgs.Message, "message", "", "Message to send (required)")
	fs.StringVar(&cliArgs.AttachmentPath, "attachment", "", "Image file to attach (png, jpeg, gif, webp; max 2.5 MiB)")
	fs.StringVar(&cliArgs.AttachmentType, "attachment-type", "", "Expected attachment MIME type (optional)")
	fs.StringVar(&cliArgs.Title, "t", "", "Message title")
	fs.StringVar(&cliArgs.Title, "title", "", "Message title")
	fs.StringVar(&cliArgs.URL, "url", "", "Supplementary URL shown with the notification")
	fs.StringVar(&cliArgs.URLTitle, "url-title", "", "Title for the supplementary URL")
	fs.BoolVar(&cliArgs.HTML, "html", false, "Render the message as HTML")
	fs.BoolVar(&cliArgs.Monospace, "monospace", false, "Render the message in a monospace font")
	fs.IntVar(&cliArgs.TTL, "ttl", 0, "Seconds until the notification is deleted (0 = unset)")
	fs.Int64Var(&cliArgs.Timestamp, "timestamp", 0, "Unix timestamp in seconds to display (0 = unset)")
	fs.IntVar(&cliArgs.Priority, "p", 0, "Priority (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)")
	fs.IntVar(&cliArgs.Priority, "priority", 0, "Priority (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)")
	fs.StringVar(&cliArgs.Sound, "s", "", "Sound name")
	fs.StringVar(&cliArgs.Sound, "sound", "", "Sound name")
	fs.IntVar(&cliArgs.ExpireTime, "e", 0, "Expire time in seconds for emergency messages (max 10800)")
	fs.IntVar(&cliArgs.ExpireTime, "expire", 0, "Expire time in seconds for emergency messages (max 10800)")
	fs.IntVar(&cliArgs.RetryTime, "r", 0, "Retry interval in seconds for emergency messages (min 30)")
	fs.IntVar(&cliArgs.RetryTime, "retry", 0, "Retry interval in seconds for emergency messages (min 30)")
	fs.StringVar(&cliArgs.DeviceName, "d", "", "Device name")
	fs.StringVar(&cliArgs.DeviceName, "device", "", "Device name")
	fs.BoolVar(&cliArgs.ShowVersion, "version", false, "Show version")
	fs.BoolVar(&cliArgs.ShowHelp, "h", false, "Show help")
	fs.BoolVar(&cliArgs.ShowHelp, "help", false, "Show help")

	// Custom usage function
	fs.Usage = func() {
		showMainHelp(fs, args[0])
	}

	if err := fs.Parse(args[1:]); err != nil {
		return nil, err
	}
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "p" || f.Name == "priority" {
			cliArgs.PrioritySet = true
		}
	})

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

// Run encapsulates the main application logic for easier testing
func Run(args []string, client PushoverClient) error {
	// Parse command-line arguments
	cliArgs, err := ParseArgs(args)
	if err != nil {
		return fmt.Errorf("argument parsing failed: %w", err)
	}

	// Handle help and version flags
	if cliArgs.ShowHelp || cliArgs.ShowVersion {
		return nil // Already handled in ParseArgs
	}
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("configuration loading failed: %w", err)
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	attachment, err := loadAttachment(context.Background(), cliArgs.AttachmentPath, "", cliArgs.AttachmentType)
	if err != nil {
		return fmt.Errorf("invalid attachment: %w", err)
	}
	cliArgs.Attachment = attachment

	// Create Pushover client if not provided (for testing)
	var recipient *pushover.Recipient
	if client == nil {
		client, recipient, err = newPushoverClient(config)
		if err != nil {
			return fmt.Errorf("failed to create Pushover client: %w", err)
		}
	} else {
		// For testing, create a dummy recipient
		recipient = pushover.NewRecipient("dummy")
	}

	// Create the message
	pushMessage, err := CreateMessage(cliArgs.Message, cliArgs.Title, config, cliArgs)
	if err != nil {
		return fmt.Errorf("invalid notification: %w", err)
	}

	// Send the notification
	response, err := client.SendMessage(pushMessage, recipient)
	if err != nil && !errors.Is(err, pushover.ErrInvalidHeaders) {
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Println(formatSendResult(response, ""))
	return nil
}
