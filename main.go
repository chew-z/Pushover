// Command pushover runs the Pushover MCP server and command-line client.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

// version is a variable, not a constant, so the Taskfile and release script
// can inject the build version with -ldflags "-X main.version=...".
var version = "dev"

// hasSubcommand reports whether args[1] names a subcommand. A value within
// Levenshtein distance 1 of a known subcommand, or with one adjacent pair
// transposed, is treated as a typo and returned as an error, so a misspelled
// subcommand is not sent as a notification message.
func hasSubcommand(args []string) (subcommand string, found bool, err error) {
	if len(args) < 2 {
		return "", false, nil
	}

	subcommand = args[1]
	switch subcommand {
	case "mcp", "cancel", "glance":
		return subcommand, true, nil
	}
	if !strings.HasPrefix(subcommand, "-") {
		for _, name := range []string{"mcp", "cancel", "glance"} {
			if levenshtein(subcommand, name) == 1 || hasAdjacentTransposition(subcommand, name) {
				return "", false, fmt.Errorf("unknown subcommand %q, did you mean %q?", subcommand, name)
			}
		}
	}
	return "", false, nil
}

// levenshtein returns the edit distance between two strings, measured in runes.
func levenshtein(a, b string) int {
	aRunes := []rune(a)
	bRunes := []rune(b)
	previous := make([]int, len(bRunes)+1)
	current := make([]int, len(bRunes)+1)

	for j := range previous {
		previous[j] = j
	}
	for i, aRune := range aRunes {
		current[0] = i + 1
		for j, bRune := range bRunes {
			cost := 0
			if aRune != bRune {
				cost = 1
			}
			current[j+1] = min(previous[j+1]+1, current[j]+1, previous[j]+cost)
		}
		previous, current = current, previous
	}

	return previous[len(bRunes)]
}

// hasAdjacentTransposition reports whether two rune strings differ by one adjacent swap.
func hasAdjacentTransposition(a, b string) bool {
	aRunes := []rune(a)
	bRunes := []rune(b)
	if len(aRunes) != len(bRunes) {
		return false
	}

	firstDifference := -1
	for i := range aRunes {
		if aRunes[i] == bRunes[i] {
			continue
		}
		if firstDifference == -1 {
			firstDifference = i
			continue
		}
		if i != firstDifference+1 || aRunes[firstDifference] != bRunes[i] || aRunes[i] != bRunes[firstDifference] {
			return false
		}
		for j := i + 1; j < len(aRunes); j++ {
			if aRunes[j] != bRunes[j] {
				return false
			}
		}
		return true
	}
	return false
}

func main() {
	// Dispatch subcommands before the default notification CLI.
	subcommand, hasSubcmd, err := hasSubcommand(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if hasSubcmd {
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
		case "glance":
			if err := runGlanceCommand(os.Args, os.Stdout, os.Stderr); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		return
	}

	// Default CLI mode
	err = Run(os.Args, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
