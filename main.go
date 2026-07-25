package main

import (
	"fmt"
	"log/slog"
	"os"

	_ "github.com/joho/godotenv/autoload"
)

const version = "1.0.0"

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

func main() {
	// Check for subcommands first
	if subcommand, hasSubcmd := hasSubcommand(os.Args); hasSubcmd {
		switch subcommand {
		case "mcp":
			// Parse MCP subcommand arguments
			if err := parseMCPArgs(os.Args); err != nil {
				slog.Error("MCP server failed", "error", err)
				os.Exit(1)
			}
			return
		}
	}

	// Default CLI mode
	err := Run(os.Args, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
