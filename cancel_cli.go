package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
)

// receiptPatternString is the regular expression used for Pushover receipts.
const receiptPatternString = "^[A-Za-z0-9]{30}$"

// receiptPattern validates Pushover emergency notification receipts.
var receiptPattern = regexp.MustCompile(receiptPatternString)

// validateReceipt verifies that receipt has Pushover's required format.
func validateReceipt(receipt string) error {
	if !receiptPattern.MatchString(receipt) {
		return errors.New("receipt must be 30 alphanumeric characters")
	}
	return nil
}

// showCancelHelp writes the cancel subcommand help text.
func showCancelHelp(out io.Writer, progName string) {
	fmt.Fprintf(out, "Usage: %s cancel -receipt RECEIPT\n", progName)
	fmt.Fprintln(out, "\nCancel retries of an emergency Pushover notification")
	fmt.Fprintln(out, "\nOptions:")
	fmt.Fprintln(out, "  -receipt string                Receipt returned when the emergency notification was sent")
	fmt.Fprintln(out, "  -h, -help                      Show help")
	fmt.Fprintln(out, "\nEnvironment Variables:")
	fmt.Fprintln(out, "  APP_KEY                        Pushover application key (required)")
	fmt.Fprintln(out, "  RECIPIENT_KEY                  Pushover recipient key (required)")
}

// runCancelCommand parses and runs the cancel subcommand. args includes the program name and "cancel".
func runCancelCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) < 2 {
		return errors.New("cancel command arguments are incomplete")
	}

	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(stderr)
	receipt := fs.String("receipt", "", "Receipt returned when the emergency notification was sent")
	showHelp := fs.Bool("h", false, "Show help")
	showHelpLong := fs.Bool("help", false, "Show help")
	fs.Usage = func() { showCancelHelp(stdout, args[0]) }
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if *showHelp || *showHelpLong {
		fs.Usage()
		return nil
	}
	if err := validateReceipt(*receipt); err != nil {
		return fmt.Errorf("invalid receipt: %w", err)
	}

	client, _, err := newPushoverClient(LoadConfig())
	if err != nil {
		return fmt.Errorf("failed to create Pushover client: %w", err)
	}
	if _, err := client.CancelEmergencyNotification(*receipt); err != nil {
		return fmt.Errorf("cancel emergency notification: %w", err)
	}

	_, _ = fmt.Fprintln(stdout, formatCancelResult(*receipt, ""))
	return nil
}
