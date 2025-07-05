package main

import (
	"flag"
	"fmt"
	"os"
)

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
