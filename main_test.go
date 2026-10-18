package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gregdel/pushover"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MockPushoverClient implements PushoverClient for testing
type MockPushoverClient struct {
	lastMessage         *pushover.Message
	lastRecipient       *pushover.Recipient
	lastReceipt         string
	lastGlance          *pushover.Glance
	lastGlanceRecipient *pushover.Recipient
	// err, when non-nil, is returned by every method.
	err error
	// response supplies the configured SendMessage response.
	response *pushover.Response
}

// SendMessage records a message send and returns the configured mock result.
func (m *MockPushoverClient) SendMessage(message *pushover.Message, recipient *pushover.Recipient) (*pushover.Response, error) {
	m.lastMessage = message
	m.lastRecipient = recipient

	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		return m.response, nil
	}

	return &pushover.Response{Status: 1}, nil
}

// CancelEmergencyNotification records an emergency cancellation and returns the configured mock result.
func (m *MockPushoverClient) CancelEmergencyNotification(receipt string) (*pushover.Response, error) {
	m.lastReceipt = receipt

	if m.err != nil {
		return nil, m.err
	}

	return &pushover.Response{Status: 1}, nil
}

// SendGlanceUpdate records a Glances update and returns the configured mock result.
func (m *MockPushoverClient) SendGlanceUpdate(glance *pushover.Glance, recipient *pushover.Recipient) (*pushover.Response, error) {
	m.lastGlance = glance
	m.lastGlanceRecipient = recipient

	if m.err != nil {
		return nil, m.err
	}

	return &pushover.Response{Status: 1}, nil
}

// swapNewPushoverClient replaces the Pushover client factory with mock for a test.
func swapNewPushoverClient(t *testing.T, mock *MockPushoverClient) {
	t.Helper()

	original := newPushoverClient
	newPushoverClient = func(Config) (PushoverClient, *pushover.Recipient, error) {
		return mock, pushover.NewRecipient("dummy"), nil
	}
	t.Cleanup(func() {
		newPushoverClient = original
	})
}

func TestCreateMessage_UseDefaultTitle(t *testing.T) {
	config := Config{
		DefaultTitle: "Default Title",
		Priority:     int(pushover.PriorityNormal),
		Sound:        pushover.SoundVibrate,
		ExpireTime:   300,
	}

	cliArgs := &CLIArgs{
		Message: "Test message",
		Title:   "", // Empty title should use default
	}

	msg, err := CreateMessage("Test message", "", config, cliArgs)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}

	if msg.Title != "Default Title" {
		t.Errorf("Expected title 'Default Title', got '%s'", msg.Title)
	}
}

func TestCreateMessage_CLIOverridesConfig(t *testing.T) {
	config := Config{
		Priority:   int(pushover.PriorityLow),
		Sound:      pushover.SoundVibrate,
		ExpireTime: 180,
	}

	cliArgs := &CLIArgs{
		Message:     "Test message",
		Priority:    int(pushover.PriorityHigh),
		PrioritySet: true,
		Sound:       pushover.SoundSiren,
		ExpireTime:  600,
	}

	msg, err := CreateMessage("Test message", "Test title", config, cliArgs)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}

	if msg.Priority != int(pushover.PriorityHigh) {
		t.Errorf("Expected priority %d, got %d", pushover.PriorityHigh, msg.Priority)
	}

	if msg.Sound != pushover.SoundSiren {
		t.Errorf("Expected sound %s, got %s", pushover.SoundSiren, msg.Sound)
	}

	expectedExpire := 600
	if int(msg.Expire.Seconds()) != expectedExpire {
		t.Errorf("Expected expire %d seconds, got %d", expectedExpire, int(msg.Expire.Seconds()))
	}
}

func TestRun_WithMockClient(t *testing.T) {
	mockClient := &MockPushoverClient{}

	// Simulate command line: pushover -m "test message" -t "test title"
	args := []string{"pushover", "-m", "test message", "-t", "test title"}

	err := Run(args, mockClient)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if mockClient.lastMessage == nil {
		t.Error("Expected message to be sent")
		return
	}

	if mockClient.lastMessage.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", mockClient.lastMessage.Message)
	}

	if mockClient.lastMessage.Title != "test title" {
		t.Errorf("Expected title 'test title', got '%s'", mockClient.lastMessage.Title)
	}
}

func TestParseArgs_ShowHelp(t *testing.T) {
	args := []string{"pushover", "-h"}

	cliArgs, err := ParseArgs(args)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !cliArgs.ShowHelp {
		t.Error("Expected ShowHelp to be true")
	}
}

func TestParseArgs_ShowVersion(t *testing.T) {
	args := []string{"pushover", "-version"}

	cliArgs, err := ParseArgs(args)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !cliArgs.ShowVersion {
		t.Error("Expected ShowVersion to be true")
	}
}

func TestParseArgs_PositionalArgs(t *testing.T) {
	args := []string{"pushover", "test message", "test title"}

	cliArgs, err := ParseArgs(args)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cliArgs.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", cliArgs.Message)
	}

	if cliArgs.Title != "test title" {
		t.Errorf("Expected title 'test title', got '%s'", cliArgs.Title)
	}
}

func TestParseArgs_MissingMessage(t *testing.T) {
	args := []string{"pushover"}

	_, err := ParseArgs(args)
	if err == nil {
		t.Error("Expected error for missing message")
	}
}

func TestNewPushoverClient_MissingKeys(t *testing.T) {
	config := Config{
		AppKey:       "",
		RecipientKey: "test",
	}

	_, _, err := NewPushoverClient(config)
	if err == nil {
		t.Error("Expected error for missing APP_KEY")
	}

	config.AppKey = "test"
	config.RecipientKey = ""

	_, _, err = NewPushoverClient(config)
	if err == nil {
		t.Error("Expected error for missing RECIPIENT_KEY")
	}
}

// --- New Tests for MCP Server Functionality ---

// newTestMCPConfig provides a valid MCPConfig for testing purposes.
func newTestMCPConfig() *MCPConfig {
	return &MCPConfig{
		PushoverAppKey:          "test_app_key",
		PushoverRecipientKey:    "test_recipient_key",
		PushoverDefaultPriority: int(pushover.PriorityNormal),
		PushoverDefaultExpire:   180,
		PushoverDefaultRetry:    60,
		HTTPAddress:             ":8080",
		HTTPPath:                "/mcp",
		AuthEnabled:             true,
		AuthSecretKey:           "a-very-secret-key-for-testing-purpose",
	}
}

func TestMCPConfig_Validation(t *testing.T) {
	testCases := []struct {
		name        string
		modifier    func(c *MCPConfig)
		expectError bool
		errContains string
	}{
		{"valid config", func(c *MCPConfig) {}, false, ""},
		{"missing app key", func(c *MCPConfig) { c.PushoverAppKey = "" }, true, "APP_KEY environment variable is required"},
		{"missing recipient key", func(c *MCPConfig) { c.PushoverRecipientKey = "" }, true, "RECIPIENT_KEY environment variable is required"},
		{"priority too low", func(c *MCPConfig) { c.PushoverDefaultPriority = -3 }, true, "PUSHOVER_PRIORITY must be between -2 and 2"},
		{"priority too high", func(c *MCPConfig) { c.PushoverDefaultPriority = 3 }, true, "PUSHOVER_PRIORITY must be between -2 and 2"},
		{"auth enabled but no secret", func(c *MCPConfig) { c.AuthSecretKey = "" }, true, "PUSHOVER_AUTH_SECRET_KEY is required"},
		{"emergency priority requires expire", func(c *MCPConfig) {
			c.PushoverDefaultPriority = int(pushover.PriorityEmergency)
			c.PushoverDefaultExpire = 0
		}, true, "PUSHOVER_EXPIRE must be > 0"},
		{"emergency priority requires retry", func(c *MCPConfig) {
			c.PushoverDefaultPriority = int(pushover.PriorityEmergency)
			c.PushoverDefaultRetry = 0
		}, true, "PUSHOVER_RETRY must be >= 30"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := newTestMCPConfig()
			tc.modifier(config)
			err := config.Validate()
			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected an error, but got none")
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tc.errContains, err)
				}
			} else if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}
		})
	}
}

func TestHttpServerEndpoints(t *testing.T) {
	config := newTestMCPConfig()
	hsm := NewHTTPServerManager(config)

	// We don't start the server, just test handlers directly
	// Create an MCP server instance to handle capability requests
	mcpServer := setupMCPServer(config)
	hsm.mcpServer = server.NewStreamableHTTPServer(mcpServer)
	hsm.tools = mcpServer

	t.Run("Health Endpoint", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()
		hsm.handleHealth(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp map[string]any
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Could not decode response: %v", err)
		}

		if resp["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", resp["status"])
		}
	})

	t.Run("Capabilities Endpoint", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/capabilities", nil)
		rr := httptest.NewRecorder()
		hsm.handleCapabilities(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp map[string]any
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Could not decode response: %v", err)
		}

		if resp["name"] != "pushover-mcp-server" {
			t.Errorf("Expected name 'pushover-mcp-server', got '%s'", resp["name"])
		}
		if !resp["authentication"].(map[string]any)["enabled"].(bool) {
			t.Error("Expected authentication to be enabled in capabilities")
		}
	})

}

func TestMCPSchemaValidation(t *testing.T) {
	config := newTestMCPConfig()
	mcpServer := setupMCPServer(config)

	c, err := client.NewInProcessClient(mcpServer)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("failed to start client: %v", err)
	}
	defer func() { _ = c.Close() }()
	if _, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "test-client", Version: "0.0.1"},
		},
	}); err != nil {
		t.Fatalf("failed to initialize client: %v", err)
	}

	cases := []struct {
		name string
		args map[string]any
	}{
		{
			name: "priority as string rejected",
			args: map[string]any{"message": "test", "priority": "5"},
		},
		{
			name: "priority out of range",
			args: map[string]any{"message": "test", "priority": 5},
		},
		{
			name: "message too long",
			args: map[string]any{"message": strings.Repeat("a", 1025)},
		},
		{
			name: "expire out of range",
			args: map[string]any{"message": "test", "expire": 99999},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := c.CallTool(ctx, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "send_notification",
					Arguments: tc.args,
				},
			})
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if !result.IsError {
				t.Errorf("expected schema validation error (IsError=true), got success")
			}
		})
	}
}

func TestNewMCPConfigFromEnv(t *testing.T) {
	t.Setenv("PUSHOVER_HTTP_ADDRESS", ":9090")
	t.Setenv("PUSHOVER_AUTH_ENABLED", "true")
	t.Setenv("PUSHOVER_AUTH_SECRET_KEY", "env-secret")
	t.Setenv("APP_KEY", "env-app-key")
	t.Setenv("RECIPIENT_KEY", "env-recipient-key")

	// Test that flag=false but env=true results in true
	config, err := NewMCPConfig(false)
	if err != nil {
		t.Fatalf("NewMCPConfig failed: %v", err)
	}

	if config.HTTPAddress != ":9090" {
		t.Errorf("Expected HTTPAddress :9090, got %s", config.HTTPAddress)
	}
	if !config.AuthEnabled {
		t.Error("Expected AuthEnabled to be true from environment")
	}
	if config.AuthSecretKey != "env-secret" {
		t.Errorf("Expected AuthSecretKey from env, but it was not set")
	}

	// Test that flag=true overrides env=false (or unset)
	os.Unsetenv("PUSHOVER_AUTH_ENABLED")
	config, err = NewMCPConfig(true)
	if err != nil {
		t.Fatalf("NewMCPConfig failed: %v", err)
	}
	if !config.AuthEnabled {
		t.Error("Expected AuthEnabled to be true from flag")
	}
}

// --- Auth Context Helper Tests ---

// --- HTTPContextFunc Tests ---

// --- extractTokenFromHeader Tests ---

// --- CORS Middleware Tests ---

// --- wrapWithAuth Tests ---

// --- handleSendNotification with authenticated context ---

// --- Run error paths ---

func TestRun_SendFailure(t *testing.T) {
	mockClient := &MockPushoverClient{err: errors.New("mock error")}
	args := []string{"pushover", "-m", "test"}

	err := Run(args, mockClient)
	if err == nil {
		t.Fatal("Expected error when send fails")
	}
	if !strings.Contains(err.Error(), "failed to send message") {
		t.Errorf("Expected send failure error, got: %v", err)
	}
}

// TestRun_InvalidHeadersStillDelivered verifies ErrInvalidHeaders is not reported as a send failure.
func TestRun_InvalidHeadersStillDelivered(t *testing.T) {
	mockClient := &MockPushoverClient{err: pushover.ErrInvalidHeaders}
	args := []string{"pushover", "-m", "test"}

	if err := Run(args, mockClient); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
}

func TestRun_ArgParseError(t *testing.T) {
	err := Run([]string{"pushover"}, nil)
	if err == nil {
		t.Fatal("Expected error for missing message")
	}
	if !strings.Contains(err.Error(), "argument parsing failed") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestRun_HelpFlag(t *testing.T) {
	err := Run([]string{"pushover", "-h"}, nil)
	if err != nil {
		t.Errorf("Expected no error for help flag, got: %v", err)
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	err := Run([]string{"pushover", "-bogus-flag", "test"}, nil)
	if err == nil {
		t.Fatal("Expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), "argument parsing failed") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestRun_ClientCreationFailure(t *testing.T) {
	t.Setenv("APP_KEY", "")
	t.Setenv("RECIPIENT_KEY", "")

	err := Run([]string{"pushover", "-m", "test"}, nil)
	if err == nil {
		t.Fatal("Expected error when client creation fails")
	}
	if !strings.Contains(err.Error(), "failed to create Pushover client") {
		t.Errorf("Expected client creation error, got: %v", err)
	}
}

// --- SendNotification ---

func TestSendNotification_Error(t *testing.T) {
	mockClient := &MockPushoverClient{err: errors.New("mock error")}
	msg := pushover.NewMessageWithTitle("test", "")
	recipient := pushover.NewRecipient("dummy")

	err := SendNotification(mockClient, msg, recipient)
	if err == nil {
		t.Error("Expected error from SendNotification")
	}
}

func TestSendNotification_Success(t *testing.T) {
	mockClient := &MockPushoverClient{}
	msg := pushover.NewMessageWithTitle("test", "")
	recipient := pushover.NewRecipient("dummy")

	err := SendNotification(mockClient, msg, recipient)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// --- CreateMessage additional coverage ---

func TestCreateMessage_DeviceAndRetry(t *testing.T) {
	config := Config{
		DeviceName: "default-device",
		RetryTime:  60,
	}
	cliArgs := &CLIArgs{
		Message:    "test",
		DeviceName: "my-phone",
		RetryTime:  120,
	}

	msg, err := CreateMessage("test", "", config, cliArgs)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}

	if msg.DeviceName != "my-phone" {
		t.Errorf("DeviceName = %q, want %q", msg.DeviceName, "my-phone")
	}
	if msg.Retry != 120*time.Second {
		t.Errorf("Retry = %v, want %v", msg.Retry, 120*time.Second)
	}
}

func TestCreateMessage_ConfigFallbacks(t *testing.T) {
	config := Config{
		DeviceName: "config-device",
		RetryTime:  45,
		Priority:   1,
	}
	cliArgs := &CLIArgs{Message: "test"}

	msg, err := CreateMessage("test", "", config, cliArgs)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}

	if msg.DeviceName != "config-device" {
		t.Errorf("DeviceName = %q, want %q", msg.DeviceName, "config-device")
	}
	if msg.Retry != 45*time.Second {
		t.Errorf("Retry = %v, want %v", msg.Retry, 45*time.Second)
	}
	if msg.Priority != 1 {
		t.Errorf("Priority = %d, want 1", msg.Priority)
	}
}

// --- HTTPServerManager.Shutdown ---

func TestShutdown_NilServer(t *testing.T) {
	hsm := &HTTPServerManager{}
	err := hsm.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Expected nil error for nil server, got: %v", err)
	}
}

// --- Health/Capabilities wrong method ---

func TestHealthEndpoint_WrongMethod(t *testing.T) {
	config := newTestMCPConfig()
	hsm := NewHTTPServerManager(config)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()
	hsm.handleHealth(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestCapabilitiesEndpoint_WrongMethod(t *testing.T) {
	config := newTestMCPConfig()
	hsm := NewHTTPServerManager(config)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/capabilities", nil)
	rr := httptest.NewRecorder()
	hsm.handleCapabilities(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

// --- GenerateTokenCommand / TokenInfo (output capture) ---

// --- parseEnvDuration Tests ---

func TestParseEnvDuration(t *testing.T) {
	t.Run("valid duration", func(t *testing.T) {
		t.Setenv("TEST_DUR", "5m")
		got := parseEnvDuration("TEST_DUR", 30*time.Second)
		if got != 5*time.Minute {
			t.Errorf("got %v, want 5m", got)
		}
	})

	t.Run("invalid duration falls back to default", func(t *testing.T) {
		t.Setenv("TEST_DUR", "notaduration")
		got := parseEnvDuration("TEST_DUR", 10*time.Second)
		if got != 10*time.Second {
			t.Errorf("got %v, want default 10s", got)
		}
	})

	t.Run("empty env falls back to default", func(t *testing.T) {
		got := parseEnvDuration("TEST_DUR_UNSET", 15*time.Second)
		if got != 15*time.Second {
			t.Errorf("got %v, want default 15s", got)
		}
	})
}

// --- parseEnvStringSlice Tests ---

func TestParseEnvStringSlice(t *testing.T) {
	t.Run("comma separated", func(t *testing.T) {
		t.Setenv("TEST_SLICE", "a, b ,c")
		got := parseEnvStringSlice("TEST_SLICE", nil)
		want := []string{"a", "b", "c"}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("empty items filtered", func(t *testing.T) {
		t.Setenv("TEST_SLICE", " , , ")
		got := parseEnvStringSlice("TEST_SLICE", []string{"default"})
		if len(got) != 1 || got[0] != "default" {
			t.Errorf("got %v, want [default]", got)
		}
	})

	t.Run("unset env returns default", func(t *testing.T) {
		got := parseEnvStringSlice("TEST_SLICE_UNSET", []string{"x"})
		if len(got) != 1 || got[0] != "x" {
			t.Errorf("got %v, want [x]", got)
		}
	})
}

// --- hasSubcommand Tests ---

func TestHasSubcommand(t *testing.T) {
	if sub, ok := hasSubcommand([]string{"pushover", "mcp"}); !ok || sub != "mcp" {
		t.Errorf("got (%q, %v), want (mcp, true)", sub, ok)
	}
	if sub, ok := hasSubcommand([]string{"pushover", "cancel"}); !ok || sub != "cancel" {
		t.Errorf("got (%q, %v), want (cancel, true)", sub, ok)
	}
	if sub, ok := hasSubcommand([]string{"pushover", "glance"}); !ok || sub != "glance" {
		t.Errorf("got (%q, %v), want (glance, true)", sub, ok)
	}
	if _, ok := hasSubcommand([]string{"pushover", "unknown"}); ok {
		t.Error("Expected false for unknown subcommand")
	}
	if _, ok := hasSubcommand([]string{"pushover"}); ok {
		t.Error("Expected false for no subcommand")
	}
}

// --- parseMCPArgs Tests ---

// --- generateTokenFromArgs Tests ---

// --- handleGenerateToken additional paths ---

// --- Structured Logging Tests ---

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{" error ", slog.LevelError},
		{"", slog.LevelInfo},
		{"bogus", slog.LevelInfo},
	}

	for _, tc := range cases {
		if got := parseLogLevel(tc.input); got != tc.want {
			t.Errorf("parseLogLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestSetupServerLogging(t *testing.T) {
	defer slog.SetDefault(slog.Default())

	t.Setenv("PUSHOVER_LOG_LEVEL", "warn")

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	setupServerLogging("http")
	slog.Info("should be filtered")
	slog.Warn("visible message", "key", "value")

	_ = w.Close()
	os.Stderr = oldStderr

	out, _ := io.ReadAll(r)
	output := string(out)

	if strings.Contains(output, "should be filtered") {
		t.Error("Info message should be filtered at warn level")
	}
	if !strings.Contains(output, "visible message") {
		t.Fatal("Warn message missing from output")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("Log output is not valid JSON: %v\noutput: %s", err, output)
	}
	if entry["service"] != "pushover-mcp" {
		t.Errorf("service = %v, want pushover-mcp", entry["service"])
	}
	if entry["transport"] != "http" {
		t.Errorf("transport = %v, want http", entry["transport"])
	}
	if entry["key"] != "value" {
		t.Errorf("key = %v, want value", entry["key"])
	}
}
