package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/gregdel/pushover"
)

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

// showMainHelp displays the main help text including subcommands
func showMainHelp(progName string) {
	fmt.Fprintf(os.Stdout, "Usage: %s [OPTIONS] | %s SUBCOMMAND [OPTIONS]\n", progName, progName)
	fmt.Fprintln(os.Stdout, "\nSend push notifications via Pushover")
	fmt.Fprintln(os.Stdout, "\nSubcommands:")
	fmt.Fprintln(os.Stdout, "  mcp               Start MCP server mode")
	fmt.Fprintln(os.Stdout, "\nOptions:")
	fmt.Fprintln(os.Stdout, "  -d string")
	fmt.Fprintln(os.Stdout, "        Device name")
	fmt.Fprintln(os.Stdout, "  -device string")
	fmt.Fprintln(os.Stdout, "        Device name")
	fmt.Fprintln(os.Stdout, "  -e int")
	fmt.Fprintln(os.Stdout, "        Expire time in seconds")
	fmt.Fprintln(os.Stdout, "  -expire int")
	fmt.Fprintln(os.Stdout, "        Expire time in seconds")
	fmt.Fprintln(os.Stdout, "  -h    Show help")
	fmt.Fprintln(os.Stdout, "  -help")
	fmt.Fprintln(os.Stdout, "        Show help")
	fmt.Fprintln(os.Stdout, "  -m string")
	fmt.Fprintln(os.Stdout, "        Message to send (required)")
	fmt.Fprintln(os.Stdout, "  -message string")
	fmt.Fprintln(os.Stdout, "        Message to send (required)")
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
	fmt.Fprintln(os.Stdout, "  -title string")
	fmt.Fprintln(os.Stdout, "        Message title")
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
		showMainHelp(args[0])
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
