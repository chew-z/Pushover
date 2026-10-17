package main

import (
	"strings"
	"testing"
	"time"

	"github.com/gregdel/pushover"
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

// TestCreateMessageTTL verifies notification lifetime validation and conversion to a duration.
func TestCreateMessageTTL(t *testing.T) {
	testCases := []struct {
		name      string
		ttl       int
		wantTTL   time.Duration
		wantError string
	}{
		{"TTL set", 60, time.Minute, ""},
		{"TTL unset", 0, 0, ""},
		{"negative TTL", -1, 0, "ttl must be at least 1 second"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, err := CreateMessage("test", "", Config{}, &CLIArgs{TTL: tc.ttl})
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("CreateMessage() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateMessage() error = %v", err)
			}
			if message.TTL != tc.wantTTL {
				t.Errorf("TTL = %v, want %v", message.TTL, tc.wantTTL)
			}
		})
	}
}

// TestCreateMessageExpireLimit verifies Pushover's maximum emergency expiration period.
func TestCreateMessageExpireLimit(t *testing.T) {
	testCases := []struct {
		name         string
		configExpire int
		cliExpire    int
		wantExpire   int
		wantError    string
	}{
		{"CLI maximum", 0, maxExpireSeconds, maxExpireSeconds, ""},
		{"CLI exceeds maximum", 0, maxExpireSeconds + 1, 0, "expire must be at most 10800 seconds"},
		{"config fallback maximum", maxExpireSeconds, 0, maxExpireSeconds, ""},
		{"config fallback exceeds maximum", maxExpireSeconds + 1, 0, 0, "expire must be at most 10800 seconds"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, err := CreateMessage("test", "", Config{ExpireTime: tc.configExpire}, &CLIArgs{ExpireTime: tc.cliExpire})
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("CreateMessage() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateMessage() error = %v", err)
			}
			if message.Expire != time.Duration(tc.wantExpire)*time.Second {
				t.Errorf("Expire = %v, want %v", message.Expire, time.Duration(tc.wantExpire)*time.Second)
			}
		})
	}
}

// TestCreateMessageTimestamp verifies optional timestamp validation and propagation.
func TestCreateMessageTimestamp(t *testing.T) {
	testCases := []struct {
		name      string
		timestamp int64
		wantError string
	}{
		{"timestamp omitted", 0, ""},
		{"timestamp set", 1700000000, ""},
		{"negative timestamp", -1, "timestamp must be a positive Unix time in seconds"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, err := CreateMessage("test", "", Config{}, &CLIArgs{Timestamp: tc.timestamp})
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("CreateMessage() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateMessage() error = %v", err)
			}
			if message.Timestamp != tc.timestamp {
				t.Errorf("Timestamp = %d, want %d", message.Timestamp, tc.timestamp)
			}
		})
	}
}

// TestFormatSendResult verifies successful send results include available response details.
func TestFormatSendResult(t *testing.T) {
	reset := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		name     string
		response *pushover.Response
		actor    string
		want     string
	}{
		{"nil response", nil, "", "Notification sent."},
		{"receipt only", &pushover.Response{Receipt: "R"}, "", "Notification sent. Receipt: R"},
		{"limit with reset", &pushover.Response{Limit: &pushover.Limit{Remaining: 7496, Total: 10000, NextReset: reset}}, "", "Notification sent. Quota: 7496/10000 remaining, resets 2026-11-01T00:00:00Z"},
		{"limit without reset", &pushover.Response{Limit: &pushover.Limit{Remaining: 7496, Total: 10000}}, "", "Notification sent. Quota: 7496/10000 remaining"},
		{"receipt limit and actor", &pushover.Response{Receipt: "R", Limit: &pushover.Limit{Remaining: 7496, Total: 10000, NextReset: reset}}, "X", "Notification sent. Receipt: R. Quota: 7496/10000 remaining, resets 2026-11-01T00:00:00Z (sent by X)"},
		{"nil response and actor", nil, "X", "Notification sent. (sent by X)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatSendResult(tc.response, tc.actor); got != tc.want {
				t.Errorf("formatSendResult() = %q, want %q", got, tc.want)
			}
		})
	}
}
