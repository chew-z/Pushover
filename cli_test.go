package main

import (
	"strings"
	"testing"
)

// TestRunPriorityOverride verifies that explicit CLI priority zero overrides the environment default.
func TestRunPriorityOverride(t *testing.T) {
	t.Setenv("PUSHOVER_PRIORITY", "-1")

	testCases := []struct {
		name string
		args []string
		want int
	}{
		{"long priority flag", []string{"pushover", "-priority", "0", "-m", "hi"}, 0},
		{"short priority flag", []string{"pushover", "-p", "0", "-m", "hi"}, 0},
		{"no priority flag", []string{"pushover", "-m", "hi"}, -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockPushoverClient{}
			if err := Run(tc.args, mock); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if mock.lastMessage == nil {
				t.Fatal("Run() did not send a message")
			}
			if mock.lastMessage.Priority != tc.want {
				t.Errorf("priority = %d, want %d", mock.lastMessage.Priority, tc.want)
			}
		})
	}
}

// TestParseArgsPrioritySet verifies that ParseArgs retains priority flag presence independently of its value.
func TestParseArgsPrioritySet(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		wantPriority    int
		wantPrioritySet bool
	}{
		{"long priority flag", []string{"pushover", "-priority", "0", "-m", "hi"}, 0, true},
		{"short priority flag", []string{"pushover", "-p", "0", "-m", "hi"}, 0, true},
		{"no priority flag", []string{"pushover", "-m", "hi"}, 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := ParseArgs(tc.args)
			if err != nil {
				t.Fatalf("ParseArgs() error = %v", err)
			}
			if args.Priority != tc.wantPriority {
				t.Errorf("Priority = %d, want %d", args.Priority, tc.wantPriority)
			}
			if args.PrioritySet != tc.wantPrioritySet {
				t.Errorf("PrioritySet = %t, want %t", args.PrioritySet, tc.wantPrioritySet)
			}
		})
	}
}

// TestRunURL verifies URL flags reach the message and invalid URL title input does not send.
func TestRunURL(t *testing.T) {
	testCases := []struct {
		name         string
		args         []string
		wantError    string
		wantURL      string
		wantURLTitle string
	}{
		{
			name:         "URL and title",
			args:         []string{"pushover", "-m", "hi", "-url", "https://x", "-url-title", "X"},
			wantURL:      "https://x",
			wantURLTitle: "X",
		},
		{
			name:      "URL title without URL",
			args:      []string{"pushover", "-m", "hi", "-url-title", "X"},
			wantError: "url_title requires url",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockPushoverClient{}
			err := Run(tc.args, mock)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("Run() error = %v, want text %q", err, tc.wantError)
				}
				if mock.lastMessage != nil {
					t.Fatalf("mock received a message: %#v", mock.lastMessage)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if mock.lastMessage == nil {
				t.Fatal("Run() did not send a message")
			}
			if mock.lastMessage.URL != tc.wantURL {
				t.Errorf("URL = %q, want %q", mock.lastMessage.URL, tc.wantURL)
			}
			if mock.lastMessage.URLTitle != tc.wantURLTitle {
				t.Errorf("URLTitle = %q, want %q", mock.lastMessage.URLTitle, tc.wantURLTitle)
			}
		})
	}
}

// TestRunTextRendering verifies rendering flags reach the message and invalid combinations do not send.
func TestRunTextRendering(t *testing.T) {
	testCases := []struct {
		name      string
		args      []string
		wantError string
		monospace bool
	}{
		{
			name:      "monospace",
			args:      []string{"pushover", "-m", "hi", "-monospace"},
			monospace: true,
		},
		{
			name:      "HTML and monospace",
			args:      []string{"pushover", "-m", "hi", "-html", "-monospace"},
			wantError: "html and monospace are mutually exclusive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockPushoverClient{}
			err := Run(tc.args, mock)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("Run() error = %v, want text %q", err, tc.wantError)
				}
				if mock.lastMessage != nil {
					t.Fatalf("mock received a message: %#v", mock.lastMessage)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if mock.lastMessage == nil {
				t.Fatal("Run() did not send a message")
			}
			if mock.lastMessage.Monospace != tc.monospace {
				t.Errorf("Monospace = %t, want %t", mock.lastMessage.Monospace, tc.monospace)
			}
		})
	}
}
