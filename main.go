package main

import (
	"fmt"
	"log/slog"
	"os"

	_ "github.com/joho/godotenv/autoload"
)

// version is a variable, not a constant, so the Taskfile and release script
// can inject the build version with -ldflags "-X main.version=...".
var version = "1.0.0"

// hasSubcommand checks if the first argument is a subcommand
func hasSubcommand(args []string) (string, bool) {
	if len(args) < 2 {
		return "", false
	}

	subcommand := args[1]
	switch subcommand {
	case "mcp", "cancel":
		return subcommand, true
	default:
		return "", false
	}
}

func main() {
	// Dispatch subcommands before the default notification CLI.
	if subcommand, hasSubcmd := hasSubcommand(os.Args); hasSubcmd {
		switch subcommand {
		case "mcp":
			if err := runMCPCommand(os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
				slog.Error("MCP server failed", "error", err)
				os.Exit(1)
			}
		case "cancel":
			if err := runCancelCommand(os.Args, os.Stdout, os.Stderr); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		return
	}

	// Default CLI mode
	err := Run(os.Args, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
