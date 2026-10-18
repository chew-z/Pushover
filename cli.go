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

// showMainHelp displays the main help text including subcommands
func showMainHelp(progName string) {
	fmt.Fprintf(os.Stdout, "Usage: %s [OPTIONS] | %s SUBCOMMAND [OPTIONS]\n", progName, progName)
	fmt.Fprintln(os.Stdout, "\nSend push notifications via Pushover")
	fmt.Fprintln(os.Stdout, "\nSubcommands:")
	fmt.Fprintln(os.Stdout, "  mcp               Start MCP server mode")
	fmt.Fprintln(os.Stdout, "\nOptions:")
	fmt.Fprintln(os.Stdout, "  -attachment string")
	fmt.Fprintln(os.Stdout, "        Image file to attach (png, jpeg, gif, webp; max 2.5 MiB)")
	fmt.Fprintln(os.Stdout, "  -attachment-type string")
	fmt.Fprintln(os.Stdout, "        Expected attachment MIME type (optional)")
	fmt.Fprintln(os.Stdout, "  -d string")
	fmt.Fprintln(os.Stdout, "        Device name")
	fmt.Fprintln(os.Stdout, "  -device string")
	fmt.Fprintln(os.Stdout, "        Device name")
	fmt.Fprintln(os.Stdout, "  -e int")
	fmt.Fprintln(os.Stdout, "        Expire time in seconds for emergency messages (max 10800)")
	fmt.Fprintln(os.Stdout, "  -expire int")
	fmt.Fprintln(os.Stdout, "        Expire time in seconds for emergency messages (max 10800)")
	fmt.Fprintln(os.Stdout, "  -h    Show help")
	fmt.Fprintln(os.Stdout, "  -help")
	fmt.Fprintln(os.Stdout, "        Show help")
	fmt.Fprintln(os.Stdout, "  -html")
	fmt.Fprintln(os.Stdout, "        Render the message as HTML")
	fmt.Fprintln(os.Stdout, "  -m string")
	fmt.Fprintln(os.Stdout, "        Message to send (required)")
	fmt.Fprintln(os.Stdout, "  -message string")
	fmt.Fprintln(os.Stdout, "        Message to send (required)")
	fmt.Fprintln(os.Stdout, "  -monospace")
	fmt.Fprintln(os.Stdout, "        Render the message in a monospace font")
	fmt.Fprintln(os.Stdout, "  -p int")
	fmt.Fprintln(os.Stdout, "        Priority (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)")
	fmt.Fprintln(os.Stdout, "  -priority int")
	fmt.Fprintln(os.Stdout, "        Priority (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)")
	fmt.Fprintln(os.Stdout, "  -s string")
	fmt.Fprintln(os.Stdout, "        Sound name")
	fmt.Fprintln(os.Stdout, "  -sound string")
	fmt.Fprintln(os.Stdout, "        Sound name")
	fmt.Fprintln(os.Stdout, "  -t string")
	fmt.Fprintln(os.Stdout, "        Message title")
	fmt.Fprintln(os.Stdout, "  -timestamp int")
	fmt.Fprintln(os.Stdout, "        Unix timestamp in seconds to display (0 = unset)")
	fmt.Fprintln(os.Stdout, "  -title string")
	fmt.Fprintln(os.Stdout, "        Message title")
	fmt.Fprintln(os.Stdout, "  -ttl int")
	fmt.Fprintln(os.Stdout, "        Seconds until the notification is deleted (0 = unset)")
	fmt.Fprintln(os.Stdout, "  -url string")
	fmt.Fprintln(os.Stdout, "        Supplementary URL shown with the notification")
	fmt.Fprintln(os.Stdout, "  -url-title string")
	fmt.Fprintln(os.Stdout, "        Title for the supplementary URL")
	fmt.Fprintln(os.Stdout, "  -version")
	fmt.Fprintln(os.Stdout, "        Show version")
	fmt.Fprintln(os.Stdout, "\nEnvironment Variables:")
	fmt.Fprintln(os.Stdout, "  APP_KEY           Pushover application key (required)")
	fmt.Fprintln(os.Stdout, "  RECIPIENT_KEY     Pushover recipient key (required)")
	fmt.Fprintln(os.Stdout, "  DEVICE_NAME       Default device name")
	fmt.Fprintln(os.Stdout, "  DEFAULT_TITLE     Default message title")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_PRIORITY Default priority")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_SOUND    Default sound")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_EXPIRE   Default expire time")
}

// ParseArgs parses command-line arguments using the flag package
func ParseArgs(args []string) (*CLIArgs, error) {
	fs := flag.NewFlagSet("pushover", flag.ContinueOnError)

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
		showMainHelp(args[0])
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
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Println(formatSendResult(response, ""))
	return nil
}
