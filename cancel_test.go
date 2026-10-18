package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestHandleCancelEmergency verifies successful emergency cancellation results.
func TestHandleCancelEmergency(t *testing.T) {
	receipt := "uQiRzpo4DXghDmr9QzzfQu27cmVRsG"
	testCases := []struct {
		name  string
		actor string
		want  string
	}{
		{name: "without actor", want: "Emergency notification uQiRzpo4DXghDmr9QzzfQu27cmVRsG canceled"},
		{name: "with actor", actor: "alice", want: "Emergency notification uQiRzpo4DXghDmr9QzzfQu27cmVRsG canceled (sent by alice)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockPushoverClient{}
			swapNewPushoverClient(t, mock)
			request := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"receipt": receipt}}}
			ctx := context.Background()
			if tc.actor != "" {
				ctx = context.WithValue(ctx, actorKey{}, tc.actor)
			}

			result, err := handleCancelEmergency(ctx, request, newTestMCPConfig())
			if err != nil {
				t.Fatalf("handleCancelEmergency() error = %v", err)
			}
			if result.IsError {
				t.Fatalf("handleCancelEmergency() result = %v, want success", result)
			}
			if mock.lastReceipt != receipt {
				t.Errorf("receipt = %q, want %q", mock.lastReceipt, receipt)
			}
			content, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("result content = %T, want mcp.TextContent", result.Content[0])
			}
			if content.Text != tc.want {
				t.Errorf("result text = %q, want %q", content.Text, tc.want)
			}
		})
	}
}

// TestHandleCancelEmergencyBadReceipt verifies invalid receipts do not reach the client.
func TestHandleCancelEmergencyBadReceipt(t *testing.T) {
	testCases := []struct {
		name    string
		receipt string
	}{
		{name: "empty", receipt: ""},
		{name: "too short", receipt: strings.Repeat("a", 29)},
		{name: "too long", receipt: strings.Repeat("a", 31)},
		{name: "non-alphanumeric", receipt: strings.Repeat("a", 29) + "-"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockPushoverClient{}
			swapNewPushoverClient(t, mock)
			request := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"receipt": tc.receipt}}}

			result, err := handleCancelEmergency(context.Background(), request, newTestMCPConfig())
			if err != nil {
				t.Fatalf("handleCancelEmergency() error = %v", err)
			}
			if !result.IsError {
				t.Fatalf("handleCancelEmergency() result = %v, want tool error", result)
			}
			if mock.lastReceipt != "" {
				t.Errorf("receipt = %q, want client not called", mock.lastReceipt)
			}
		})
	}
}

// TestHandleCancelEmergencyAPIError verifies client failures return a tool error.
func TestHandleCancelEmergencyAPIError(t *testing.T) {
	mock := &MockPushoverClient{shouldError: true}
	swapNewPushoverClient(t, mock)
	receipt := strings.Repeat("a", 30)
	request := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"receipt": receipt}}}

	result, err := handleCancelEmergency(context.Background(), request, newTestMCPConfig())
	if err != nil {
		t.Fatalf("handleCancelEmergency() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("handleCancelEmergency() result = %v, want tool error", result)
	}
	if !strings.Contains(fmt.Sprint(result), "Failed to cancel") {
		t.Errorf("result = %v, want cancellation error", result)
	}
}

// TestRunCancelCommand verifies the cancel CLI command uses the Pushover client.
func TestRunCancelCommand(t *testing.T) {
	mock := &MockPushoverClient{}
	swapNewPushoverClient(t, mock)
	receipt := "uQiRzpo4DXghDmr9QzzfQu27cmVRsG"
	var stdout, stderr bytes.Buffer

	err := runCancelCommand([]string{"pushover", "cancel", "-receipt", receipt}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runCancelCommand() error = %v", err)
	}
	if want := "Emergency notification uQiRzpo4DXghDmr9QzzfQu27cmVRsG canceled\n"; stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
	if mock.lastReceipt != receipt {
		t.Errorf("receipt = %q, want %q", mock.lastReceipt, receipt)
	}
}

// TestRunCancelCommandBadReceipt verifies invalid CLI receipts do not reach the client.
func TestRunCancelCommandBadReceipt(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{name: "missing receipt", args: []string{"pushover", "cancel"}},
		{name: "malformed receipt", args: []string{"pushover", "cancel", "-receipt", "invalid"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockPushoverClient{}
			swapNewPushoverClient(t, mock)
			var stdout, stderr bytes.Buffer

			err := runCancelCommand(tc.args, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), "invalid receipt") {
				t.Fatalf("runCancelCommand() error = %v, want invalid receipt", err)
			}
			if mock.lastReceipt != "" {
				t.Errorf("receipt = %q, want client not called", mock.lastReceipt)
			}
		})
	}
}

// TestRunCancelCommandAPIError verifies CLI client failures are wrapped.
func TestRunCancelCommandAPIError(t *testing.T) {
	mock := &MockPushoverClient{shouldError: true}
	swapNewPushoverClient(t, mock)
	var stdout, stderr bytes.Buffer

	err := runCancelCommand([]string{"pushover", "cancel", "-receipt", strings.Repeat("a", 30)}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "cancel emergency notification") {
		t.Fatalf("runCancelCommand() error = %v, want cancellation error", err)
	}
}

// TestRunCancelCommandHelp verifies the cancel CLI help output.
func TestRunCancelCommandHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := runCancelCommand([]string{"pushover", "cancel", "-h"}, &stdout, &stderr); err != nil {
		t.Fatalf("runCancelCommand() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "cancel -receipt") {
		t.Errorf("stdout = %q, want cancel usage", stdout.String())
	}
}
