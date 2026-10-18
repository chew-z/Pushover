package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/gregdel/pushover"
)

// showGlanceHelp writes the glance subcommand help text.
func showGlanceHelp(out io.Writer, progName string) {
	fmt.Fprintf(out, "Usage: %s glance [OPTIONS]\n", progName)
	fmt.Fprintln(out, "\nSend an Apple Watch complication Glance update")
	fmt.Fprintln(out, "\nUse only for Apple Watch complications. Wait at least 20 minutes between updates, send at most 50 per day, and allow up to 10 minutes for an update to appear. Pass an empty string to clear a text field.")
	fmt.Fprintln(out, "\nOptions:")
	fmt.Fprintln(out, "  -title string                  Glance title")
	fmt.Fprintln(out, "  -text string                   Main Glance text")
	fmt.Fprintln(out, "  -subtext string                Secondary Glance text")
	fmt.Fprintln(out, "  -count int                     Integer count")
	fmt.Fprintln(out, "  -percent int                   Integer percentage from 0 through 100")
	fmt.Fprintln(out, "  -device string                 Target device name")
	fmt.Fprintln(out, "  -h, -help                      Show help")
	fmt.Fprintln(out, "\nEnvironment Variables:")
	fmt.Fprintln(out, "  APP_KEY                        Pushover application key (required)")
	fmt.Fprintln(out, "  RECIPIENT_KEY                  Pushover recipient key (required)")
}

// runGlanceCommand parses and runs the glance subcommand. args includes the program name and "glance".
func runGlanceCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) < 2 {
		return errors.New("glance command arguments are incomplete")
	}

	fs := flag.NewFlagSet("glance", flag.ContinueOnError)
	fs.SetOutput(stderr)
	title := fs.String("title", "", "Glance title")
	text := fs.String("text", "", "Main Glance text")
	subtext := fs.String("subtext", "", "Secondary Glance text")
	count := fs.Int("count", 0, "Integer count")
	percent := fs.Int("percent", 0, "Integer percentage from 0 through 100")
	device := fs.String("device", "", "Target device name")
	showHelp := fs.Bool("h", false, "Show help")
	showHelpLong := fs.Bool("help", false, "Show help")
	fs.Usage = func() { showGlanceHelp(stdout, args[0]) }
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if *showHelp || *showHelpLong {
		fs.Usage()
		return nil
	}

	arguments := make(map[string]any)
	fs.Visit(func(current *flag.Flag) {
		switch current.Name {
		case "title":
			arguments["title"] = *title
		case "text":
			arguments["text"] = *text
		case "subtext":
			arguments["subtext"] = *subtext
		case "count":
			arguments["count"] = *count
		case "percent":
			arguments["percent"] = *percent
		}
	})
	glance, err := buildGlance(arguments, *device)
	if err != nil {
		return fmt.Errorf("invalid Glance: %w", err)
	}

	client, recipient, err := newPushoverClient(LoadConfig())
	if err != nil {
		return fmt.Errorf("failed to create Pushover client: %w", err)
	}
	response, err := client.SendGlanceUpdate(glance, recipient)
	if err != nil && !errors.Is(err, pushover.ErrInvalidHeaders) {
		return fmt.Errorf("send Glance update: %w", err)
	}

	_, _ = fmt.Fprintln(stdout, formatGlanceResult(response, ""))
	return nil
}
