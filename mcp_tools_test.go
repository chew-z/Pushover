package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestHandleSendNotification verifies send_notification results without reaching Pushover.
func TestHandleSendNotification(t *testing.T) {
	testCases := []struct {
		name            string
		arguments       map[string]any
		defaultPriority int
		shouldError     bool
		wantError       bool
		errorText       string
		wantMessage     string
		wantPriority    int
		wantURL         string
		wantURLTitle    string
		wantHTML        bool
		wantMonospace   bool
		wantNoMessage   bool
	}{
		{
			name:        "successful send",
			arguments:   map[string]any{"message": "test message", "title": "test title"},
			wantMessage: "test message",
		},
		{
			name:      "missing message",
			arguments: map[string]any{"title": "test"},
			wantError: true,
			errorText: "Message parameter is required",
		},
		{
			name:        "mock send failure",
			arguments:   map[string]any{"message": "test message"},
			shouldError: true,
			wantError:   true,
			errorText:   "Failed to send notification",
		},
		{
			name:            "explicit priority zero",
			arguments:       map[string]any{"message": "test message", "priority": 0},
			defaultPriority: -1,
			wantMessage:     "test message",
			wantPriority:    0,
		},
		{
			name:            "omitted priority uses config default",
			arguments:       map[string]any{"message": "test message"},
			defaultPriority: -1,
			wantMessage:     "test message",
			wantPriority:    -1,
		},
		{
			name:         "URL and URL title",
			arguments:    map[string]any{"message": "test message", "url": "https://example.com", "url_title": "Example"},
			wantMessage:  "test message",
			wantURL:      "https://example.com",
			wantURLTitle: "Example",
		},
		{
			name:          "URL title requires URL",
			arguments:     map[string]any{"message": "test message", "url_title": "Example"},
			wantError:     true,
			errorText:     "url_title requires url",
			wantNoMessage: true,
		},
		{
			name:        "HTML",
			arguments:   map[string]any{"message": "test message", "html": true},
			wantMessage: "test message",
			wantHTML:    true,
		},
		{
			name:          "HTML and monospace are mutually exclusive",
			arguments:     map[string]any{"message": "test message", "html": true, "monospace": true},
			wantError:     true,
			errorText:     "html and monospace are mutually exclusive",
			wantNoMessage: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockPushoverClient{shouldError: tc.shouldError}
			swapNewPushoverClient(t, mock)

			config := newTestMCPConfig()
			if tc.defaultPriority != 0 {
				config.PushoverDefaultPriority = tc.defaultPriority
			}
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: tc.arguments},
			}
			result, err := handleSendNotification(context.Background(), request, config)
			if err != nil {
				t.Fatalf("handleSendNotification() error = %v", err)
			}
			if result == nil {
				t.Fatal("handleSendNotification() returned nil result")
			}
			if result.IsError != tc.wantError {
				t.Fatalf("result.IsError = %t, want %t", result.IsError, tc.wantError)
			}
			if tc.errorText != "" && !strings.Contains(fmt.Sprint(result), tc.errorText) {
				t.Fatalf("result = %v, want text %q", result, tc.errorText)
			}
			if tc.wantNoMessage {
				if mock.lastMessage != nil {
					t.Fatalf("mock received a message: %#v", mock.lastMessage)
				}
				return
			}
			if tc.wantMessage != "" {
				if mock.lastMessage == nil {
					t.Fatal("mock did not receive a message")
				}
				if mock.lastMessage.Message != tc.wantMessage {
					t.Errorf("message = %q, want %q", mock.lastMessage.Message, tc.wantMessage)
				}
				if mock.lastMessage.Priority != tc.wantPriority {
					t.Errorf("priority = %d, want %d", mock.lastMessage.Priority, tc.wantPriority)
				}
				if mock.lastMessage.URL != tc.wantURL {
					t.Errorf("URL = %q, want %q", mock.lastMessage.URL, tc.wantURL)
				}
				if mock.lastMessage.URLTitle != tc.wantURLTitle {
					t.Errorf("URLTitle = %q, want %q", mock.lastMessage.URLTitle, tc.wantURLTitle)
				}
				if mock.lastMessage.HTML != tc.wantHTML {
					t.Errorf("HTML = %t, want %t", mock.lastMessage.HTML, tc.wantHTML)
				}
				if mock.lastMessage.Monospace != tc.wantMonospace {
					t.Errorf("Monospace = %t, want %t", mock.lastMessage.Monospace, tc.wantMonospace)
				}
			}
		})
	}
}
