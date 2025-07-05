package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gregdel/pushover"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

// hasSubcommand checks if the first argument is a subcommand
func hasSubcommand(args []string) (string, bool) {
	if len(args) < 2 {
		return "", false
	}

	subcommand := args[1]
	switch subcommand {
	case "mcp":
		return subcommand, true
	default:
		return "", false
	}
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

// showMCPHelp displays the MCP subcommand help text
func showMCPHelp(progName string) {
	fmt.Fprintf(os.Stdout, "Usage: %s mcp [OPTIONS]\n", progName)
	fmt.Fprintln(os.Stdout, "\nStart MCP server for push notifications")
	fmt.Fprintln(os.Stdout, "\nOptions:")
	fmt.Fprintln(os.Stdout, "  -auth-enabled")
	fmt.Fprintln(os.Stdout, "        Enable JWT authentication for HTTP transport")
	fmt.Fprintln(os.Stdout, "  -generate-token")
	fmt.Fprintln(os.Stdout, "        Generate a JWT token and exit")
	fmt.Fprintln(os.Stdout, "  -h    Show help")
	fmt.Fprintln(os.Stdout, "  -help")
	fmt.Fprintln(os.Stdout, "        Show help")
	fmt.Fprintln(os.Stdout, "  -token-expiration int")
	fmt.Fprintln(os.Stdout, "        Token expiration in hours (default: 744 = 31 days) (default 744)")
	fmt.Fprintln(os.Stdout, "  -token-role string")
	fmt.Fprintln(os.Stdout, "        Role for token generation (default \"admin\")")
	fmt.Fprintln(os.Stdout, "  -token-user-id string")
	fmt.Fprintln(os.Stdout, "        User ID for token generation (default \"user1\")")
	fmt.Fprintln(os.Stdout, "  -token-username string")
	fmt.Fprintln(os.Stdout, "        Username for token generation (default \"admin\")")
	fmt.Fprintln(os.Stdout, "  -transport string")
	fmt.Fprintln(os.Stdout, "        Transport mode: 'stdio' (default) or 'http' (default \"stdio\")")
	fmt.Fprintln(os.Stdout, "\nEnvironment Variables:")
	fmt.Fprintln(os.Stdout, "  APP_KEY                        Pushover application key (required)")
	fmt.Fprintln(os.Stdout, "  RECIPIENT_KEY                  Pushover recipient key (required)")
	fmt.Fprintln(os.Stdout, "  DEVICE_NAME                    Default device name")
	fmt.Fprintln(os.Stdout, "  DEFAULT_TITLE                  Default message title")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_PRIORITY              Default priority")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_SOUND                 Default sound")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_EXPIRE                Default expire time")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_HTTP_ADDRESS          Server bind address (default \":8080\")")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_HTTP_PATH             Endpoint path (default \"/mcp\")")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_HTTP_STATELESS        Session management mode (default false)")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_HTTP_HEARTBEAT        Heartbeat interval (default \"30s\")")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_HTTP_TIMEOUT          Request timeout (default \"30s\")")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_HTTP_CORS_ENABLED     Enable CORS (default true)")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_HTTP_CORS_ORIGINS     Allowed origins (default \"*\")")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_AUTH_ENABLED          Enable JWT authentication (default false)")
	fmt.Fprintln(os.Stdout, "  PUSHOVER_AUTH_SECRET_KEY       JWT signing secret")
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

// parseMCPArgs parses MCP subcommand arguments
func parseMCPArgs(args []string) error {
	// Create a new flag set for MCP subcommand
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)

	// MCP-specific flags
	var (
		authEnabled     = fs.Bool("auth-enabled", false, "Enable JWT authentication for HTTP transport")
		generateToken   = fs.Bool("generate-token", false, "Generate a JWT token and exit")
		tokenExpiration = fs.Int("token-expiration", 744, "Token expiration in hours (default: 744 = 31 days)")
		tokenRole       = fs.String("token-role", "admin", "Role for token generation")
		tokenUserID     = fs.String("token-user-id", "user1", "User ID for token generation")
		tokenUsername   = fs.String("token-username", "admin", "Username for token generation")
		transportMode   = fs.String("transport", "stdio", "Transport mode: 'stdio' (default) or 'http'")
		showHelp        = fs.Bool("h", false, "Show help")
		showHelp2       = fs.Bool("help", false, "Show help")
	)

	// Custom usage function for MCP
	fs.Usage = func() {
		showMCPHelp(args[0])
	}

	// Parse arguments, skipping the program name and "mcp" subcommand
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}

	// Handle help flag
	if *showHelp || *showHelp2 {
		fs.Usage()
		return nil
	}

	// Set global flags from local variables
	// We need to update the global variables that the MCP functions use
	if *authEnabled {
		// Set environment variable to enable auth
		os.Setenv("PUSHOVER_AUTH_ENABLED", "true")
	}

	// Handle token generation
	if *generateToken {
		return generateTokenFromArgs(*tokenUserID, *tokenUsername, *tokenRole, *tokenExpiration)
	}

	// Run MCP server with the specified transport
	return runMCPServerWithTransport(*transportMode, *authEnabled)
}

// generateTokenFromArgs generates a JWT token using the provided arguments
func generateTokenFromArgs(userID, username, role string, expirationHours int) error {
	config, err := NewMCPConfig(false)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if config.AuthSecretKey == "" {
		return fmt.Errorf("PUSHOVER_AUTH_SECRET_KEY environment variable is required for token generation")
	}

	authMiddleware := NewAuthMiddleware(config.AuthSecretKey, true)
	token, err := authMiddleware.GenerateJWT(userID, username, role, expirationHours)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	fmt.Printf("Generated JWT token:\n%s\n", token)
	fmt.Printf("\nTo use this token, include it in the Authorization header:\n")
	fmt.Printf("Authorization: Bearer %s\n", token)
	fmt.Printf("\nToken details:\n")
	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Role: %s\n", role)
	fmt.Printf("Expires in: %d hours\n", expirationHours)
	return nil
}

// runMCPServerWithTransport starts the MCP server with the specified transport mode
func runMCPServerWithTransport(transport string, authEnabled bool) error {
	// Load MCP configuration
	config, err := NewMCPConfig(authEnabled)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create MCP server
	mcpServer := setupMCPServer(config)

	switch transport {
	case "stdio":
		log.Println("Starting MCP server with STDIO transport...")
		return server.ServeStdio(mcpServer)

	case "http":
		log.Printf("Starting MCP server with HTTP transport on %s%s...", config.HTTPAddress, config.HTTPPath)
		httpManager := NewHTTPServerManager(config)
		return httpManager.Start(mcpServer)

	default:
		return fmt.Errorf("unsupported transport: %s (supported: stdio, http)", transport)
	}
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

// MCP Server Functions

// setupMCPServer creates and configures the MCP server with tools
func setupMCPServer(config *MCPConfig) *server.MCPServer {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"pushover-mcp-server",
		version,
	)

	// Add the send_notification tool
	mcpServer.AddTool(
		mcp.NewTool("send_notification",
			mcp.WithDescription("Send a notification via Pushover"),
			mcp.WithString("message",
				mcp.Description("The notification message to send (required)"),
				mcp.Required(),
			),
			mcp.WithString("title",
				mcp.Description("Optional title for the notification"),
			),
			mcp.WithString("priority",
				mcp.Description("Priority level (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)"),
			),
			mcp.WithString("device",
				mcp.Description("Target device name"),
			),
			mcp.WithString("sound",
				mcp.Description("Notification sound name"),
			),
			mcp.WithString("expire",
				mcp.Description("Expiration time in seconds for emergency messages"),
			),
		),
		wrapWithAuth(handleSendNotification, "send_notification", config),
	)

	return mcpServer
}

// wrapWithAuth wraps tool handlers with authentication and logging
func wrapWithAuth(
	handler func(context.Context, mcp.CallToolRequest, *MCPConfig) (*mcp.CallToolResult, error),
	toolName string,
	config *MCPConfig,
) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("Calling tool '%s'...", toolName)

		// Check authentication for HTTP requests
		if isHTTPRequest(ctx) && config.AuthEnabled {
			if !isAuthenticated(ctx) {
				authError := getAuthError(ctx)
				return mcp.NewToolResultError(fmt.Sprintf("Authentication required: %s", authError)), nil
			}
			log.Printf("Tool '%s' called by user: %s", toolName, getUsername(ctx))
		}

		// Call the actual handler
		result, err := handler(ctx, req, config)

		// Log result
		if err != nil {
			log.Printf("Tool '%s' failed: %v", toolName, err)
		} else {
			log.Printf("Tool '%s' completed successfully", toolName)
		}

		return result, err
	}
}

// handleSendNotification handles the send_notification tool
func handleSendNotification(ctx context.Context, request mcp.CallToolRequest, config *MCPConfig) (*mcp.CallToolResult, error) {
	// Extract required message parameter
	message, err := request.RequireString("message")
	if err != nil {
		return mcp.NewToolResultError("Message parameter is required"), nil
	}

	// Validate message length
	if len(message) > 1024 {
		return mcp.NewToolResultError("Message too long (max 1024 characters)"), nil
	}

	// Extract optional parameters
	title := request.GetString("title", "")
	priorityStr := request.GetString("priority", "")
	device := request.GetString("device", "")
	sound := request.GetString("sound", "")
	expireStr := request.GetString("expire", "")

	// Parse priority
	priority := config.PushoverDefaultPriority
	if priorityStr != "" {
		if p, err := strconv.Atoi(priorityStr); err == nil {
			if p < -2 || p > 2 {
				return mcp.NewToolResultError("Priority must be between -2 and 2"), nil
			}
			priority = p
		} else {
			return mcp.NewToolResultError("Invalid priority value"), nil
		}
	}

	// Parse expire time
	expireTime := config.PushoverDefaultExpire
	if expireStr != "" {
		if e, err := strconv.Atoi(expireStr); err == nil {
			expireTime = e
		} else {
			return mcp.NewToolResultError("Invalid expire value"), nil
		}
	}

	// Create legacy config for compatibility
	legacyConfig := config.ToLegacyConfig()

	// Create Pushover client
	client, recipient, err := NewPushoverClient(legacyConfig)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create Pushover client: %v", err)), nil
	}

	// Create CLI args structure for message creation
	cliArgs := &CLIArgs{
		Message:    message,
		Title:      title,
		Priority:   priority,
		Sound:      sound,
		ExpireTime: expireTime,
		DeviceName: device,
	}

	// Create and send message
	pushMessage := CreateMessage(message, title, legacyConfig, cliArgs)

	response, err := client.SendMessage(pushMessage, recipient)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send notification: %v", err)), nil
	}

	// Prepare response message
	responseText := "Notification sent successfully"
	if response != nil && response.Receipt != "" {
		responseText += fmt.Sprintf(". Receipt: %s", response.Receipt)
	}

	// Add user info if authenticated
	if isHTTPRequest(ctx) && isAuthenticated(ctx) {
		username := getUsername(ctx)
		if username != "" {
			responseText += fmt.Sprintf(" (sent by %s)", username)
		}
	}

	return mcp.NewToolResultText(responseText), nil
}

func main() {
	// Check for subcommands first
	if subcommand, hasSubcmd := hasSubcommand(os.Args); hasSubcmd {
		switch subcommand {
		case "mcp":
			// Configure logging for server mode
			log.SetFlags(log.LstdFlags)

			// Parse MCP subcommand arguments
			if err := parseMCPArgs(os.Args); err != nil {
				log.Printf("MCP error: %v", err)
				os.Exit(1)
			}
			return
		}
	}

	// Default CLI mode
	log.SetFlags(0) // Clean output without timestamps for CLI mode
	err := Run(os.Args, nil)
	if err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
