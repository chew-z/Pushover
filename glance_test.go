package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gregdel/pushover"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestBuildGlancePreservesOmittedAndEmptyStrings verifies omitted text is nil while explicit empty text clears it.
func TestBuildGlancePreservesOmittedAndEmptyStrings(t *testing.T) {
	omitted, err := buildGlance(map[string]any{"count": 0}, "")
	if err != nil {
		t.Fatalf("buildGlance() error = %v", err)
	}
	if omitted.Text != nil {
		t.Errorf("omitted text = %v, want nil", omitted.Text)
	}

	empty, err := buildGlance(map[string]any{"text": ""}, "")
	if err != nil {
		t.Fatalf("buildGlance() error = %v", err)
	}
	if empty.Text == nil || *empty.Text != "" {
		t.Errorf("empty text = %v, want pointer to empty string", empty.Text)
	}
}

// TestBuildGlanceValidation verifies required fields, percentage bounds, and integral numeric inputs.
func TestBuildGlanceValidation(t *testing.T) {
	testCases := []struct {
		name      string
		arguments map[string]any
		wantError string
	}{
		{name: "no fields", arguments: map[string]any{}, wantError: "at least one"},
		{name: "negative percent", arguments: map[string]any{"percent": -1}, wantError: "between 0 and 100"},
		{name: "percent over 100", arguments: map[string]any{"percent": 101}, wantError: "between 0 and 100"},
		{name: "fractional count", arguments: map[string]any{"count": 1.5}, wantError: "count must be an integer"},
		{name: "non-numeric count", arguments: map[string]any{"count": "one"}, wantError: "count must be an integer"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := buildGlance(tc.arguments, ""); err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("buildGlance() error = %v, want %q", err, tc.wantError)
			}
		})
	}

	for _, percent := range []int{0, 100} {
		glance, err := buildGlance(map[string]any{"percent": percent}, "watch")
		if err != nil {
			t.Fatalf("buildGlance(percent=%d) error = %v", percent, err)
		}
		if glance.Percent == nil || *glance.Percent != percent || glance.DeviceName != "watch" {
			t.Errorf("buildGlance(percent=%d) = %+v", percent, glance)
		}
	}
}

// TestRunGlanceCommandSendsOnlyGivenFlags verifies the CLI preserves explicit fields and does not synthesize omitted ones.
func TestRunGlanceCommandSendsOnlyGivenFlags(t *testing.T) {
	mock := &MockPushoverClient{}
	swapNewPushoverClient(t, mock)
	var stdout, stderr bytes.Buffer

	err := runGlanceCommand([]string{"pushover", "glance", "-text", "", "-count", "0", "-device", "watch"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runGlanceCommand() error = %v", err)
	}
	if mock.lastGlance == nil {
		t.Fatal("Glance was not sent")
	}
	if mock.lastGlance.Text == nil || *mock.lastGlance.Text != "" {
		t.Errorf("text = %v, want pointer to empty string", mock.lastGlance.Text)
	}
	if mock.lastGlance.Count == nil || *mock.lastGlance.Count != 0 {
		t.Errorf("count = %v, want pointer to 0", mock.lastGlance.Count)
	}
	if mock.lastGlance.Title != nil || mock.lastGlance.Subtext != nil || mock.lastGlance.Percent != nil {
		t.Errorf("omitted fields = %+v, want nil", mock.lastGlance)
	}
	if mock.lastGlance.DeviceName != "watch" {
		t.Errorf("device = %q, want watch", mock.lastGlance.DeviceName)
	}
	if !strings.Contains(stdout.String(), "Glance sent. Quota: unknown") {
		t.Errorf("stdout = %q, want unknown quota", stdout.String())
	}
}

// TestGlanceCapabilitiesListSendGlance verifies capability discovery advertises the Glance tool.
func TestGlanceCapabilitiesListSendGlance(t *testing.T) {
	tools := capabilityTools(setupMCPServer(newTestMCPConfig()))
	for _, tool := range tools {
		if tool["name"] == "send_glance" {
			return
		}
	}
	t.Error("capabilityTools() does not include send_glance")
}

// TestHandleSendGlanceRealLibraryResponses verifies real-library Glance responses and quota handling.
func TestHandleSendGlanceRealLibraryResponses(t *testing.T) {
	testCases := []struct {
		name        string
		status      int
		headers     map[string]string
		body        string
		wantError   bool
		wantContent string
	}{
		{name: "successful response without quota headers", status: http.StatusOK, body: `{"status":1}`, wantContent: "Glance sent. Quota: unknown"},
		{name: "successful response with quota headers", status: http.StatusOK, headers: map[string]string{"X-Limit-App-Limit": "7500", "X-Limit-App-Remaining": "6000", "X-Limit-App-Reset": "1393653600"}, body: `{"status":1}`, wantContent: "Glance sent. Quota: 6000/7500 remaining"},
		{name: "API error", status: http.StatusBadRequest, body: `{"status":0,"errors":["invalid"]}`, wantError: true, wantContent: "invalid"},
		{name: "server error", status: http.StatusInternalServerError, body: `{"status":0}`, wantError: true, wantContent: "http error"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Path != "/glances.json" {
					t.Errorf("URL path = %q, want /glances.json", request.URL.Path)
				}
				for name, value := range tc.headers {
					w.Header().Set(name, value)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			originalEndpoint := pushover.APIEndpoint
			pushover.APIEndpoint = server.URL
			t.Cleanup(func() { pushover.APIEndpoint = originalEndpoint })

			originalClient := newPushoverClient
			newPushoverClient = NewPushoverClient
			t.Cleanup(func() { newPushoverClient = originalClient })
			config := newTestMCPConfig()
			config.PushoverAppKey = strings.Repeat("a", 30)
			config.PushoverRecipientKey = strings.Repeat("b", 30)
			request := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"text": "status"}}}

			result, err := handleSendGlance(context.Background(), request, config)
			if err != nil {
				t.Fatalf("handleSendGlance() error = %v", err)
			}
			if result.IsError != tc.wantError {
				t.Fatalf("handleSendGlance() IsError = %t, want %t", result.IsError, tc.wantError)
			}
			encoded, err := json.Marshal(result.Content)
			if err != nil {
				t.Fatalf("marshal result content: %v", err)
			}
			if !strings.Contains(string(encoded), tc.wantContent) {
				t.Errorf("result content = %s, want %q", encoded, tc.wantContent)
			}
		})
	}
}
