package main

import (
	"strings"
	"testing"
)

// TestCreateMessageURL verifies supplementary URL validation and message fields.
func TestCreateMessageURL(t *testing.T) {
	testCases := []struct {
		name      string
		url       string
		urlTitle  string
		wantError string
	}{
		{"URL and title", "https://example.com", "Example", ""},
		{"URL only", "https://example.com", "", ""},
		{"title requires URL", "", "Example", "url_title requires url"},
		{"URL exceeds rune limit", strings.Repeat("é", 513), "", "url must be at most 512 characters"},
		{"URL at rune limit", strings.Repeat("é", 512), "", ""},
		{"URL title exceeds limit", "https://example.com", strings.Repeat("x", 101), "url_title must be at most 100 characters"},
		{"URL title at limit", "https://example.com", strings.Repeat("x", 100), ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, err := CreateMessage("test", "", Config{}, &CLIArgs{URL: tc.url, URLTitle: tc.urlTitle})
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("CreateMessage() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateMessage() error = %v", err)
			}
			if message.URL != tc.url {
				t.Errorf("URL = %q, want %q", message.URL, tc.url)
			}
			if message.URLTitle != tc.urlTitle {
				t.Errorf("URLTitle = %q, want %q", message.URLTitle, tc.urlTitle)
			}
		})
	}
}

// TestCreateMessageTextRendering verifies HTML and monospace message rendering options.
func TestCreateMessageTextRendering(t *testing.T) {
	testCases := []struct {
		name      string
		html      bool
		monospace bool
		wantError string
	}{
		{"HTML only", true, false, ""},
		{"monospace only", false, true, ""},
		{"HTML and monospace", true, true, "html and monospace are mutually exclusive"},
		{"neither", false, false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, err := CreateMessage("test", "", Config{}, &CLIArgs{HTML: tc.html, Monospace: tc.monospace})
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("CreateMessage() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateMessage() error = %v", err)
			}
			if message.HTML != tc.html {
				t.Errorf("HTML = %t, want %t", message.HTML, tc.html)
			}
			if message.Monospace != tc.monospace {
				t.Errorf("Monospace = %t, want %t", message.Monospace, tc.monospace)
			}
		})
	}
}
