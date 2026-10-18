package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chew-z/Pushover/internal/oauth"
	"golang.org/x/term"
)

// showMCPHelp writes the MCP subcommand help text to out.
func showMCPHelp(fs *flag.FlagSet, out io.Writer, progName string) {
	fmt.Fprintf(out, "Usage: %s mcp [OPTIONS]\n", progName)
	fmt.Fprintln(out, "\nStart MCP server for push notifications")
	fmt.Fprintln(out, "\nOptions:")
	fs.SetOutput(out)
	fs.PrintDefaults()
	fmt.Fprintln(out, "\nEnvironment Variables:")
	fmt.Fprintln(out, "  APP_KEY                        Pushover application key (required to run the server)")
	fmt.Fprintln(out, "  RECIPIENT_KEY                  Pushover recipient key (required to run the server)")
	fmt.Fprintln(out, "  DEVICE_NAME, DEFAULT_TITLE, PUSHOVER_PRIORITY, PUSHOVER_SOUND, PUSHOVER_EXPIRE, PUSHOVER_RETRY")
	fmt.Fprintln(out, "  PUSHOVER_HTTP_ADDRESS          Server bind address (default \"127.0.0.1:8080\")")
	fmt.Fprintln(out, "  PUSHOVER_HTTP_PATH             Endpoint path (default \"/mcp\")")
	fmt.Fprintln(out, "  PUSHOVER_HTTP_STATELESS, PUSHOVER_HTTP_HEARTBEAT, PUSHOVER_HTTP_TIMEOUT")
	fmt.Fprintln(out, "  PUSHOVER_HTTP_CORS_ENABLED     Enable CORS (default true)")
	fmt.Fprintln(out, "  PUSHOVER_HTTP_CORS_ORIGINS     Allowed origins (default empty: none)")
	fmt.Fprintln(out, "  PUSHOVER_AUTH_ENABLED, PUSHOVER_AUTH_SECRET_KEY")
	fmt.Fprintln(out, "  PUSHOVER_MCP_OAUTH, PUSHOVER_MCP_ISSUER, PUSHOVER_MCP_RESOURCE")
	fmt.Fprintln(out, "  PUSHOVER_MCP_LOGIN_USER, PUSHOVER_MCP_LOGIN_PASSWORD_HASH")
	fmt.Fprintln(out, "  PUSHOVER_MCP_ACCESS_TTL, PUSHOVER_MCP_REFRESH_TTL")
	fmt.Fprintln(out, "  PUSHOVER_LOG_LEVEL             Log level: debug, info, warn, error (default \"info\")")
}

// readPassword provides a test seam, suppressing terminal echo and reading one line from a pipe.
var readPassword = func(stdin io.Reader, stderr io.Writer) (string, error) {
	if file, ok := stdin.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		value, err := term.ReadPassword(int(file.Fd()))
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		fmt.Fprintln(stderr)
		return string(value), nil
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// runMCPCommand parses the mcp subcommand. args includes the program name and "mcp".
func runMCPCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) < 2 {
		return errors.New("mcp command arguments are incomplete")
	}
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	authEnabled := fs.Bool("auth-enabled", false, "Enable bearer authentication for HTTP transport")
	transportMode := fs.String("transport", "stdio", "Transport mode: 'stdio' (default) or 'http'")
	showHelp := fs.Bool("h", false, "Show help")
	showHelpLong := fs.Bool("help", false, "Show help")
	generateToken := fs.Bool("generate-token", false, "Generate a static bearer token and exit")
	tokenUsername := fs.String("token-username", "admin", "Token username")
	tokenTTL := fs.Duration("token-ttl", 744*time.Hour, "Token lifetime")
	hashPassword := fs.Bool("hash-password", false, "Hash an OAuth login password and exit")
	fs.Usage = func() { showMCPHelp(fs, stdout, args[0]) }
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if *showHelp || *showHelpLong {
		fs.Usage()
		return nil
	}
	if *generateToken && *hashPassword {
		return errors.New("-generate-token and -hash-password cannot be used together")
	}
	if *generateToken {
		secret := os.Getenv("PUSHOVER_AUTH_SECRET_KEY")
		if secret == "" {
			return errors.New("PUSHOVER_AUTH_SECRET_KEY environment variable is required for token generation")
		}
		token, err := GenerateStaticToken(secret, *tokenUsername, *tokenTTL)
		if err != nil {
			return fmt.Errorf("generate static token: %w", err)
		}
		_, _ = fmt.Fprintln(stdout, token)
		return nil
	}
	if *hashPassword {
		_, _ = fmt.Fprint(stderr, "Password: ")
		password, err := readPassword(stdin, stderr)
		if err != nil {
			return err
		}
		if password == "" {
			return errors.New("password must not be empty")
		}
		hash, err := oauth.HashPassword(password)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		_, _ = fmt.Fprintln(stdout, hash)
		return nil
	}
	return runMCPServerWithTransport(*transportMode, *authEnabled)
}
